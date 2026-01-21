// Package service 提供业务逻辑
package service

import (
	"sync"
	"time"

	"glance-bilibil/internal/config"
	"glance-bilibil/internal/logger"
	"glance-bilibil/internal/models"
	"glance-bilibil/internal/platform"
	"glance-bilibil/internal/worker"
)

// cacheEntry 缓存条目
type cacheEntry struct {
	videos    models.VideoList
	updatedAt time.Time
}

// VideoService 视频服务
type VideoService struct {
	client     *platform.BilibiliClient
	config     *config.Config
	cache      map[string]cacheEntry
	mu         sync.RWMutex
	workerPool *worker.Pool
}

// NewVideoService 创建视频服务
func NewVideoService(cfg *config.Config) *VideoService {
	client := platform.NewBilibiliClient()

	// 创建 Worker Pool，默认 10 个 Worker
	pool := worker.NewPool(10)
	pool.Start()

	return &VideoService{
		client:     client,
		config:     cfg,
		cache:      make(map[string]cacheEntry),
		workerPool: pool,
	}
}

// Initialize 初始化服务
func (s *VideoService) Initialize() error {
	return s.client.Initialize()
}

// fetchTask 获取单个频道视频的任务
type fetchTask struct {
	service         *VideoService
	channel         config.ChannelInfo
	limit           int
	cacheTTLSeconds int
	resultChan      chan<- models.VideoList
	wg              *sync.WaitGroup
}

// Execute 实现 worker.Task 接口
func (t *fetchTask) Execute() error {
	defer t.wg.Done()

	// 1. 尝试从缓存获取
	cacheKey := t.channel.Mid
	t.service.mu.RLock()
	entry, exists := t.service.cache[cacheKey]
	t.service.mu.RUnlock()

	// 如果缓存存在且未过期，直接使用
	if exists && time.Since(entry.updatedAt) < time.Duration(t.cacheTTLSeconds)*time.Second {
		logger.Debugw("命中有效缓存",
			"up_name", t.channel.Name,
			"up_mid", t.channel.Mid,
			"cached", true,
		)
		t.resultChan <- entry.videos
		return nil
	}

	// 2. 缓存不存在或已过期，从 API 获取
	videos, err := t.service.client.FetchUserVideos(t.channel.Mid, t.limit, t.channel.Name)
	if err != nil {
		logger.Warnw("获取视频失败",
			"up_name", t.channel.Name,
			"up_mid", t.channel.Mid,
			"error", err,
		)
		// 容错降级：如果 API 失败且有旧缓存，返回旧缓存
		if exists {
			logger.Infow("API 失败，返回过期缓存数据",
				"up_name", t.channel.Name,
				"cached", true,
			)
			t.resultChan <- entry.videos
		}
		return err
	}

	// 3. 更新缓存
	t.service.mu.Lock()
	t.service.cache[cacheKey] = cacheEntry{
		videos:    videos,
		updatedAt: time.Now(),
	}
	t.service.mu.Unlock()

	logger.Infow("获取视频成功",
		"up_name", t.channel.Name,
		"up_mid", t.channel.Mid,
		"video_count", len(videos),
		"cached", false,
	)
	t.resultChan <- videos
	return nil
}

// FetchAllVideos 并发获取所有 UP 主的视频并按时间排序
// cacheTTLSeconds 缓存有效期（秒）
func (s *VideoService) FetchAllVideos(limit int, cacheTTLSeconds int) (models.VideoList, error) {
	if len(s.config.Channels) == 0 {
		return models.VideoList{}, nil
	}

	// 创建结果通道和同步等待组
	videoChan := make(chan models.VideoList, len(s.config.Channels))
	var submitWg sync.WaitGroup  // 用于等待任务提交
	var executeWg sync.WaitGroup // 用于等待任务执行

	// 提交任务到 Worker Pool
	for _, channel := range s.config.Channels {
		submitWg.Add(1)
		executeWg.Add(1)

		ch := channel // 避免闭包问题
		task := &fetchTask{
			service:         s,
			channel:         ch,
			limit:           limit,
			cacheTTLSeconds: cacheTTLSeconds,
			resultChan:      videoChan,
			wg:              &executeWg,
		}

		go func() {
			defer submitWg.Done()
			s.workerPool.Submit(task)
		}()
	}

	// 等待所有任务提交完成
	submitWg.Wait()

	// 等待所有任务执行完成后关闭通道
	go func() {
		executeWg.Wait()
		close(videoChan)
	}()

	// 收集结果
	var allVideos models.VideoList
	for videos := range videoChan {
		allVideos = append(allVideos, videos...)
	}

	// 检查是否全部失败且无缓存
	if len(allVideos) == 0 {
		return nil, nil
	}

	// 按时间排序并限制数量
	return allVideos.SortByNewest().Limit(limit), nil
}

// FetchChannelVideos 获取单个 UP 主的视频
func (s *VideoService) FetchChannelVideos(mid string, limit int, cacheTTLSeconds int) (models.VideoList, error) {
	// 1. 尝试从缓存获取
	s.mu.RLock()
	entry, exists := s.cache[mid]
	s.mu.RUnlock()

	if exists && time.Since(entry.updatedAt) < time.Duration(cacheTTLSeconds)*time.Second {
		return entry.videos.SortByNewest().Limit(limit), nil
	}

	// 2. 从 API 获取
	videos, err := s.client.FetchUserVideos(mid, limit, "")
	if err != nil {
		if exists {
			return entry.videos.SortByNewest().Limit(limit), nil
		}
		return nil, err
	}

	// 3. 更新缓存
	s.mu.Lock()
	s.cache[mid] = cacheEntry{
		videos:    videos,
		updatedAt: time.Now(),
	}
	s.mu.Unlock()

	return videos.SortByNewest().Limit(limit), nil
}

// GetConfig 获取配置
func (s *VideoService) GetConfig() *config.Config {
	return s.config
}

// Shutdown 关闭服务（优雅关闭 Worker Pool）
func (s *VideoService) Shutdown() {
	if s.workerPool != nil {
		s.workerPool.Stop()
	}
}
