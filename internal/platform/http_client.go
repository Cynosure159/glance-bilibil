// Package platform 提供 HTTP 客户端
package platform

import (
	"net/http"
	"sync"
	"time"

	"github.com/go-resty/resty/v2"
)

var (
	// 全局单例 Resty 客户端
	restyClient     *resty.Client
	restyClientOnce sync.Once
)

// GetRestyClient 获取全局单例 Resty 客户端
func GetRestyClient() *resty.Client {
	restyClientOnce.Do(func() {
		// 自定义 HTTP Transport 配置连接池
		transport := &http.Transport{
			MaxIdleConns:        100,              // 最大空闲连接数
			MaxIdleConnsPerHost: 10,               // 每个 Host 最大空闲连接数
			IdleConnTimeout:     90 * time.Second, // 空闲连接超时
			DisableCompression:  false,            // 启用压缩
			DisableKeepAlives:   false,            // 启用 Keep-Alive
		}

		// 创建底层 HTTP 客户端
		httpClient := &http.Client{
			Timeout:   30 * time.Second, // 增加超时时间以适应较慢网络
			Transport: transport,
		}

		// 创建 Resty 客户端
		restyClient = resty.NewWithClient(httpClient)

		// 配置重试策略
		restyClient.
			SetRetryCount(3).                     // 重试次数
			SetRetryWaitTime(1 * time.Second).    // 重试等待时间
			SetRetryMaxWaitTime(5 * time.Second). // 最大等待时间
			AddRetryCondition(func(r *resty.Response, err error) bool {
				// 基于状态码的重试条件
				if err != nil {
					return true // 网络错误重试
				}
				// 5xx 服务器错误重试
				if r.StatusCode() >= 500 {
					return true
				}
				// 429 Too Many Requests 重试
				if r.StatusCode() == 429 {
					return true
				}
				return false
			})

		// 设置通用 Header
		restyClient.SetHeaders(map[string]string{
			"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
			"Accept":     "application/json",
		})
	})

	return restyClient
}
