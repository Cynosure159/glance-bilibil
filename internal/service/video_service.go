// Package service 提供业务逻辑。
package service

import (
	"math/rand"
	"sync"
	"time"

	"glance-bilibili/internal/config"
	"glance-bilibili/internal/logger"
	"glance-bilibili/internal/models"
	"glance-bilibili/internal/platform"
	"glance-bilibili/internal/worker"
)

const (
	defaultWorkerCount = 4
	cacheFetchLimit    = 50
	requestJitterMin   = 250 * time.Millisecond
	requestJitterMax   = 1200 * time.Millisecond
	cacheStaleReason   = "缓存缺失或已过期"
)

// cacheEntry 是单个 UP 主的视频缓存。
type cacheEntry struct {
	videos    models.VideoList
	updatedAt time.Time
}

// VideoService 管理视频缓存及后台刷新任务。
type VideoService struct {
	client     *platform.BilibiliClient
	config     *config.Config
	cache      map[string]cacheEntry
	refreshing map[string]bool
	mu         sync.RWMutex
	workerPool *worker.Pool
	stopCh     chan struct{}
	startOnce  sync.Once
	stopOnce   sync.Once
}

// NewVideoService 创建视频服务。
func NewVideoService(cfg *config.Config) *VideoService {
	pool := worker.NewPool(defaultWorkerCount)
	pool.Start()

	return &VideoService{
		client:     platform.NewBilibiliClient(),
		config:     cfg,
		cache:      make(map[string]cacheEntry),
		refreshing: make(map[string]bool),
		workerPool: pool,
		stopCh:     make(chan struct{}),
	}
}

// Initialize 初始化服务。
func (s *VideoService) Initialize() error {
	return s.client.Initialize()
}

// StartCacheRefresh 立即预热配置频道，并按 refresh_interval 定时刷新。
func (s *VideoService) StartCacheRefresh() {
	s.startOnce.Do(func() {
		interval := s.config.GetRefreshInterval()
		s.refreshConfiguredChannels("启动预热")
		go func() {
			ticker := time.NewTicker(interval)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					s.refreshConfiguredChannels("定时刷新")
				case <-s.stopCh:
					return
				}
			}
		}()
		logger.Infow("后台缓存刷新已启动", "refresh_interval", interval.String())
	})
}

func (s *VideoService) refreshConfiguredChannels(reason string) {
	logger.Infow("开始批量刷新配置频道缓存",
		"reason", reason,
		"channel_count", len(s.config.Channels),
	)
	for _, channel := range s.config.Channels {
		s.scheduleRefresh(channel.Mid, channel.Name, reason)
	}
}

func (s *VideoService) getCachedVideos(mid string, cacheTTL time.Duration) (models.VideoList, bool, time.Time) {
	s.mu.RLock()
	entry, exists := s.cache[mid]
	s.mu.RUnlock()
	if !exists {
		return nil, false, time.Time{}
	}
	return entry.videos, time.Since(entry.updatedAt) < cacheTTL, entry.updatedAt
}

func (s *VideoService) scheduleRefresh(mid, authorName, reason string) {
	s.mu.Lock()
	if s.refreshing[mid] {
		s.mu.Unlock()
		return
	}
	s.refreshing[mid] = true
	s.mu.Unlock()
	logger.Debugw("已调度视频缓存刷新",
		"up_mid", mid,
		"up_name", authorName,
		"reason", reason,
	)

	// Submit 在队列繁忙时可能等待，因此放到 goroutine 中，确保 HTTP 请求只读缓存。
	go s.workerPool.Submit(&refreshTask{service: s, mid: mid, authorName: authorName})
}

func (s *VideoService) finishRefresh(mid string) {
	s.mu.Lock()
	delete(s.refreshing, mid)
	s.mu.Unlock()
}

func (s *VideoService) setCachedVideos(mid string, videos models.VideoList) {
	s.mu.Lock()
	s.cache[mid] = cacheEntry{videos: videos, updatedAt: time.Now()}
	s.mu.Unlock()
}

// refreshTask 在 Worker Pool 中刷新单个频道缓存。
type refreshTask struct {
	service    *VideoService
	mid        string
	authorName string
}

// Execute 实现 worker.Task。
func (t *refreshTask) Execute() error {
	defer t.service.finishRefresh(t.mid)

	// 打散同一批预热或定时刷新请求，降低同时访问 B 站接口触发风控的概率。
	time.Sleep(randomRequestDelay())
	videos, err := t.service.client.FetchUserVideos(t.mid, cacheFetchLimit, t.authorName)
	if err != nil {
		logger.Warnw("后台刷新视频缓存失败", "up_mid", t.mid, "up_name", t.authorName, "error", err)
		return err
	}
	t.service.setCachedVideos(t.mid, videos)
	logger.Infow("后台刷新视频缓存成功", "up_mid", t.mid, "up_name", t.authorName, "video_count", len(videos))
	return nil
}

// FetchAllVideos 从缓存汇总配置频道的视频。过期或缺失的缓存会在后台刷新。
func (s *VideoService) FetchAllVideos(limit, cacheTTLSeconds int) (models.VideoList, error) {
	var allVideos models.VideoList
	var refreshChannels []string
	var cachedChannels []string
	cacheTTL := time.Duration(cacheTTLSeconds) * time.Second
	for _, channel := range s.config.Channels {
		videos, fresh, cachedAt := s.getCachedVideos(channel.Mid, cacheTTL)
		s.logCachedResponse(channel.Mid, channel.Name, cachedAt, fresh)
		if !fresh {
			refreshChannels = append(refreshChannels, formatChannel(channel))
			s.scheduleRefresh(channel.Mid, channel.Name, cacheStaleReason)
		} else {
			cachedChannels = append(cachedChannels, formatChannel(channel))
		}
		allVideos = append(allVideos, videos...)
	}
	logCacheCheck(refreshChannels, cachedChannels)
	return allVideos.SortByNewest().Limit(limit), nil
}

// FetchChannelVideos 从缓存返回单个 UP 主视频。临时 MID 仅按需刷新，不加入定时任务。
func (s *VideoService) FetchChannelVideos(mid string, limit, cacheTTLSeconds int) (models.VideoList, error) {
	cacheTTL := time.Duration(cacheTTLSeconds) * time.Second
	videos, fresh, cachedAt := s.getCachedVideos(mid, cacheTTL)
	authorName := s.channelName(mid)
	s.logCachedResponse(mid, authorName, cachedAt, fresh)
	channel := formatChannel(config.ChannelInfo{Mid: mid, Name: authorName})
	var refreshChannels, cachedChannels []string
	if !fresh {
		refreshChannels = []string{channel}
		s.scheduleRefresh(mid, authorName, cacheStaleReason)
	} else {
		cachedChannels = []string{channel}
	}
	logCacheCheck(refreshChannels, cachedChannels)
	return videos.SortByNewest().Limit(limit), nil
}

func logCacheCheck(refreshChannels, cachedChannels []string) {
	logger.Infow("请求检查视频缓存",
		"refresh_channels", refreshChannels,
		"cached_channels", cachedChannels,
	)
}

func (s *VideoService) logCachedResponse(mid, authorName string, cachedAt time.Time, fresh bool) {
	if cachedAt.IsZero() {
		return
	}
	logger.Debugw("使用视频缓存响应请求",
		"up_mid", mid,
		"up_name", authorName,
		"cache_updated_at", cachedAt,
		"cache_age", time.Since(cachedAt).Round(time.Second).String(),
		"cache_fresh", fresh,
	)
}

func formatChannel(channel config.ChannelInfo) string {
	if channel.Name == "" {
		return channel.Mid
	}
	return channel.Name + "(" + channel.Mid + ")"
}

func randomRequestDelay() time.Duration {
	window := requestJitterMax - requestJitterMin
	if window <= 0 {
		return requestJitterMin
	}
	return requestJitterMin + time.Duration(rand.Int63n(int64(window)))
}

func (s *VideoService) channelName(mid string) string {
	for _, channel := range s.config.Channels {
		if channel.Mid == mid {
			return channel.Name
		}
	}
	return ""
}

// GetConfig 获取配置。
func (s *VideoService) GetConfig() *config.Config {
	return s.config
}

// Shutdown 停止后台刷新及 Worker Pool。
func (s *VideoService) Shutdown() {
	s.stopOnce.Do(func() {
		logger.Info("停止后台缓存刷新")
		close(s.stopCh)
		if s.workerPool != nil {
			s.workerPool.Stop()
		}
	})
}
