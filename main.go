// Package main 是程序入口
package main

import (
	"embed"
	"flag"
	"fmt"
	"net/http"
	"os"

	"glance-bilibil/internal/api"
	"glance-bilibil/internal/config"
	"glance-bilibil/internal/logger"
	"glance-bilibil/internal/service"
)

//go:embed templates/*.html
var templatesFS embed.FS

func main() {
	// 初始化日志系统
	logger.AutoInit()
	defer logger.Sync()

	// 解析命令行参数
	configPath := flag.String("config", "", "配置文件路径")
	port := flag.Int("port", 8082, "HTTP 服务端口")
	limit := flag.Int("limit", 25, "默认显示视频数量")
	flag.Parse()

	// 加载配置
	cfg, err := config.Load(*configPath)
	if err != nil {
		logger.Fatalw("加载配置失败", "error", err)
	}

	logger.Infow("配置加载成功",
		"up_count", len(cfg.Channels),
	)

	// 创建服务
	svc := service.NewVideoService(cfg)

	// 初始化（获取 WBI 密钥等）
	logger.Info("正在初始化...")
	if err := svc.Initialize(); err != nil {
		logger.Warnw("初始化警告 (将在首次请求时重试)",
			"error", err,
		)
	} else {
		logger.Info("初始化成功")
	}

	// 创建处理器 (默认展示样式固定为 horizontal-cards)
	handler, err := api.NewHandler(svc, templatesFS, *limit)
	if err != nil {
		logger.Fatalw("创建处理器失败", "error", err)
	}

	// 注册路由
	http.HandleFunc("/json", handler.JSONHandler)
	http.HandleFunc("/health", handler.HealthHandler)
	http.HandleFunc("/help", handler.HelpHandler)
	http.HandleFunc("/", handler.VideosHandler)

	// 启动服务
	addr := fmt.Sprintf(":%d", *port)
	logger.Infow("服务启动",
		"address", fmt.Sprintf("http://localhost%s", addr),
		"port", *port,
	)

	if err := http.ListenAndServe(addr, nil); err != nil {
		logger.Errorw("服务器错误", "error", err)
		os.Exit(1)
	}
}
