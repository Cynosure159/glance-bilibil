// Package worker Worker Pool 单元测试
package worker

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// mockTask 用于测试的模拟任务
type mockTask struct {
	id        int
	executed  bool
	mu        sync.Mutex
	execDelay time.Duration
	shouldErr bool
	execCount *int32 // 用于并发计数
}

func (t *mockTask) Execute() error {
	if t.execDelay > 0 {
		time.Sleep(t.execDelay)
	}

	t.mu.Lock()
	t.executed = true
	t.mu.Unlock()

	if t.execCount != nil {
		atomic.AddInt32(t.execCount, 1)
	}

	if t.shouldErr {
		return errors.New("mock error")
	}
	return nil
}

func (t *mockTask) wasExecuted() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.executed
}

// TestNewPool 测试创建 Worker Pool
func TestNewPool(t *testing.T) {
	tests := []struct {
		name        string
		workerCount int
		expected    int
	}{
		{
			name:        "正常数量",
			workerCount: 5,
			expected:    5,
		},
		{
			name:        "默认数量（0 时使用默认值）",
			workerCount: 0,
			expected:    10,
		},
		{
			name:        "负数（使用默认值）",
			workerCount: -1,
			expected:    10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pool := NewPool(tt.workerCount)

			if pool.workerCount != tt.expected {
				t.Errorf("workerCount = %d, want %d", pool.workerCount, tt.expected)
			}

			// 验证队列容量（Worker 数量的 2 倍）
			expectedCap := tt.expected * 2
			if cap(pool.taskQueue) != expectedCap {
				t.Errorf("taskQueue capacity = %d, want %d", cap(pool.taskQueue), expectedCap)
			}
		})
	}
}

// TestPool_SubmitAndExecute 测试任务提交和执行
func TestPool_SubmitAndExecute(t *testing.T) {
	pool := NewPool(3)
	pool.Start()

	var execCount int32
	tasks := make([]*mockTask, 5)

	for i := 0; i < 5; i++ {
		tasks[i] = &mockTask{
			id:        i,
			execCount: &execCount,
		}
		pool.Submit(tasks[i])
	}

	// 等待任务执行
	time.Sleep(100 * time.Millisecond)
	pool.Stop()

	// 验证所有任务都被执行
	if atomic.LoadInt32(&execCount) != 5 {
		t.Errorf("执行任务数 = %d, want 5", execCount)
	}

	for i, task := range tasks {
		if !task.wasExecuted() {
			t.Errorf("任务 %d 未被执行", i)
		}
	}
}

// concurrentTask 可追踪并发度的任务
type concurrentTask struct {
	execDelay         time.Duration
	currentConcurrent *int32
	maxConcurrent     *int32
	mu                *sync.Mutex
	wg                *sync.WaitGroup
}

func (t *concurrentTask) Execute() error {
	atomic.AddInt32(t.currentConcurrent, 1)
	defer atomic.AddInt32(t.currentConcurrent, -1)

	t.mu.Lock()
	if *t.currentConcurrent > *t.maxConcurrent {
		*t.maxConcurrent = *t.currentConcurrent
	}
	t.mu.Unlock()

	time.Sleep(t.execDelay)
	t.wg.Done()
	return nil
}

// TestPool_ConcurrentExecution 测试并发执行
func TestPool_ConcurrentExecution(t *testing.T) {
	pool := NewPool(10)
	pool.Start()

	var maxConcurrent int32
	var currentConcurrent int32
	var mu sync.Mutex

	// 创建 50 个任务，每个任务执行 10ms
	taskCount := 50
	var wg sync.WaitGroup

	for i := 0; i < taskCount; i++ {
		wg.Add(1)
		task := &concurrentTask{
			execDelay:         10 * time.Millisecond,
			currentConcurrent: &currentConcurrent,
			maxConcurrent:     &maxConcurrent,
			mu:                &mu,
			wg:                &wg,
		}

		pool.Submit(task)
	}

	wg.Wait()
	pool.Stop()

	// 验证最大并发数不超过 Worker 数量
	if maxConcurrent > 10 {
		t.Errorf("最大并发数 = %d, 超过了 Worker 数量 10", maxConcurrent)
	}
}

// TestPool_ErrorHandling 测试错误处理
func TestPool_ErrorHandling(t *testing.T) {
	pool := NewPool(2)
	pool.Start()

	var execCount int32

	// 提交一个会失败的任务和一个成功的任务
	failTask := &mockTask{id: 1, shouldErr: true, execCount: &execCount}
	successTask := &mockTask{id: 2, shouldErr: false, execCount: &execCount}

	pool.Submit(failTask)
	pool.Submit(successTask)

	time.Sleep(100 * time.Millisecond)
	pool.Stop()

	// 两个任务都应该被执行（即使一个失败了）
	if atomic.LoadInt32(&execCount) != 2 {
		t.Errorf("执行任务数 = %d, want 2", execCount)
	}
}

// TestPool_StartOnce 测试 Start 只能调用一次
func TestPool_StartOnce(t *testing.T) {
	pool := NewPool(3)

	// 多次调用 Start
	pool.Start()
	pool.Start()
	pool.Start()

	// 提交任务验证 Pool 正常工作
	var execCount int32
	task := &mockTask{execCount: &execCount}
	pool.Submit(task)

	time.Sleep(50 * time.Millisecond)
	pool.Stop()

	if atomic.LoadInt32(&execCount) != 1 {
		t.Error("Pool 应该正常执行任务")
	}
}

// TestPool_WaitWithCallback 测试带回调的等待
func TestPool_WaitWithCallback(t *testing.T) {
	pool := NewPool(2)
	pool.Start()

	var execCount int32
	for i := 0; i < 3; i++ {
		pool.Submit(&mockTask{execCount: &execCount, execDelay: 10 * time.Millisecond})
	}

	callbackExecuted := false
	pool.WaitWithCallback(func() {
		callbackExecuted = true
	})

	if !callbackExecuted {
		t.Error("回调函数应该被执行")
	}

	if atomic.LoadInt32(&execCount) != 3 {
		t.Errorf("执行任务数 = %d, want 3", execCount)
	}
}
