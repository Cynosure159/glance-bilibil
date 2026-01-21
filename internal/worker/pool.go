// Package worker 提供 Worker Pool 实现
package worker

import (
	"log"
	"sync"
)

// Task 任务接口
type Task interface {
	Execute() error
}

// Pool Worker Pool 结构
type Pool struct {
	workerCount int
	taskQueue   chan Task
	wg          sync.WaitGroup
	once        sync.Once
}

// NewPool 创建新的 Worker Pool
// workerCount: Worker 数量，推荐 10-50
func NewPool(workerCount int) *Pool {
	if workerCount <= 0 {
		workerCount = 10 // 默认 10 个 Worker
	}

	return &Pool{
		workerCount: workerCount,
		taskQueue:   make(chan Task, workerCount*2), // 缓冲队列，容量为 Worker 数量的 2 倍
	}
}

// Start 启动 Worker Pool
func (p *Pool) Start() {
	p.once.Do(func() {
		log.Printf("[INFO] Worker Pool 启动，Worker 数量: %d", p.workerCount)

		for i := 0; i < p.workerCount; i++ {
			p.wg.Add(1)
			go p.worker(i)
		}
	})
}

// worker 单个 Worker 的执行逻辑
func (p *Pool) worker(id int) {
	defer p.wg.Done()

	for task := range p.taskQueue {
		if err := task.Execute(); err != nil {
			log.Printf("[WARN] Worker %d 执行任务失败: %v", id, err)
		}
	}
}

// Submit 提交任务到 Worker Pool
func (p *Pool) Submit(task Task) {
	p.taskQueue <- task
}

// Stop 停止 Worker Pool
func (p *Pool) Stop() {
	close(p.taskQueue)
	p.wg.Wait()
	log.Printf("[INFO] Worker Pool 已停止")
}

// WaitWithCallback 等待所有任务完成后执行回调
func (p *Pool) WaitWithCallback(callback func()) {
	close(p.taskQueue)
	p.wg.Wait()
	if callback != nil {
		callback()
	}
}
