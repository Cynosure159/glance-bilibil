// Package service 提供业务逻辑
package service

import (
	"log"
	"sync"

	"glance-bilibil/internal/config"
	"glance-bilibil/internal/models"
	"glance-bilibil/internal/platform"
)

// VideoService 视频服务
type VideoService struct {
	client *platform.BilibiliClient
	config *config.Config
}

// NewVideoService 创建视频服务
func NewVideoService(cfg *config.Config) *VideoService {
	client := platform.NewBilibiliClient()
	return &VideoService{
		client: client,
		config: cfg,
	}
}

// Initialize 初始化服务
func (s *VideoService) Initialize() error {
	return s.client.Initialize()
}

// FetchAllVideos 并发获取所有 UP 主的视频并按时间排序
func (s *VideoService) FetchAllVideos(limit int) (models.VideoList, error) {
	if len(s.config.Channels) == 0 {
		return models.VideoList{}, nil
	}

	// 使用默认 limit
	if limit <= 0 {
		limit = s.config.Limit
	}

	// 并发获取
	var wg sync.WaitGroup
	videoChan := make(chan models.VideoList, len(s.config.Channels))
	errChan := make(chan error, len(s.config.Channels))

	for _, channel := range s.config.Channels {
		wg.Add(1)
		go func(ch config.ChannelInfo) {
			defer wg.Done()

			// 每个 UP 主都获取 limit 个视频，以确保排序后的全局前 limit 个视频是准确的
			videos, err := s.client.FetchUserVideos(ch.Mid, limit, ch.Name)
			if err != nil {
				log.Printf("[WARN] 获取 %s (%s) 视频失败: %v", ch.Name, ch.Mid, err)
				errChan <- err
				return
			}

			log.Printf("[INFO] 获取 %s (%s) 视频成功: %d 个", ch.Name, ch.Mid, len(videos))
			videoChan <- videos
		}(channel)
	}

	// 等待所有请求完成
	wg.Wait()
	close(videoChan)
	close(errChan)

	// 收集结果
	var allVideos models.VideoList
	for videos := range videoChan {
		allVideos = append(allVideos, videos...)
	}

	// 检查是否全部失败
	if len(allVideos) == 0 {
		// 返回第一个错误
		for err := range errChan {
			return nil, err
		}
	}

	// 按时间排序并限制数量
	return allVideos.SortByNewest().Limit(limit), nil
}

// FetchChannelVideos 获取单个 UP 主的视频
func (s *VideoService) FetchChannelVideos(mid string, limit int) (models.VideoList, error) {
	if limit <= 0 {
		limit = s.config.Limit
	}

	videos, err := s.client.FetchUserVideos(mid, limit, "")
	if err != nil {
		return nil, err
	}

	return videos.SortByNewest(), nil
}

// GetConfig 获取配置
func (s *VideoService) GetConfig() *config.Config {
	return s.config
}
