// Package service 提供业务逻辑
package service

import (
	"log"
	"sync"
	"time"

	"glance-bilibil/internal/config"
	"glance-bilibil/internal/models"
	"glance-bilibil/internal/platform"
)

// cacheEntry 缓存条目
type cacheEntry struct {
	videos    models.VideoList
	updatedAt time.Time
}

// VideoService 视频服务
type VideoService struct {
	client *platform.BilibiliClient
	config *config.Config
	cache  map[string]cacheEntry
	mu     sync.RWMutex
}

// NewVideoService 创建视频服务
func NewVideoService(cfg *config.Config) *VideoService {
	client := platform.NewBilibiliClient()
	return &VideoService{
		client: client,
		config: cfg,
		cache:  make(map[string]cacheEntry),
	}
}

// Initialize 初始化服务
func (s *VideoService) Initialize() error {
	return s.client.Initialize()
}

// FetchAllVideos 并发获取所有 UP 主的视频并按时间排序
// cacheTTLSeconds 缓存有效期（秒）
func (s *VideoService) FetchAllVideos(limit int, cacheTTLSeconds int) (models.VideoList, error) {
	if len(s.config.Channels) == 0 {
		return models.VideoList{}, nil
	}

	// 并发获取
	var wg sync.WaitGroup
	videoChan := make(chan models.VideoList, len(s.config.Channels))

	for _, channel := range s.config.Channels {
		wg.Add(1)
		go func(ch config.ChannelInfo) {
			defer wg.Done()

			// 1. 尝试从缓存获取
			cacheKey := ch.Mid
			s.mu.RLock()
			entry, exists := s.cache[cacheKey]
			s.mu.RUnlock()

			// 如果缓存存在且未过期，直接使用
			if exists && time.Since(entry.updatedAt) < time.Duration(cacheTTLSeconds)*time.Second {
				log.Printf("[DEBUG] %s 命中有效缓存", ch.Name)
				videoChan <- entry.videos
				return
			}

			// 2. 缓存不存在或已过期，从 API 获取
			videos, err := s.client.FetchUserVideos(ch.Mid, limit, ch.Name)
			if err != nil {
				log.Printf("[WARN] 获取 %s (%s) 视频失败: %v", ch.Name, ch.Mid, err)
				// 容错降级：如果 API 失败且有旧缓存，返回旧缓存
				if exists {
					log.Printf("[INFO] %s API 失败，返回过期缓存数据", ch.Name)
					videoChan <- entry.videos
				}
				return
			}

			// 3. 更新缓存
			s.mu.Lock()
			s.cache[cacheKey] = cacheEntry{
				videos:    videos,
				updatedAt: time.Now(),
			}
			s.mu.Unlock()

			log.Printf("[INFO] 获取 %s (%s) 视频成功: %d 个", ch.Name, ch.Mid, len(videos))
			videoChan <- videos
		}(channel)
	}

	// 等待所有请求完成
	wg.Wait()
	close(videoChan)

	// 收集结果
	var allVideos models.VideoList
	for videos := range videoChan {
		allVideos = append(allVideos, videos...)
	}

	// 检查是否全部失败且无缓存
	if len(allVideos) == 0 {
		return nil, nil // 或者定义一个特定的错误
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
