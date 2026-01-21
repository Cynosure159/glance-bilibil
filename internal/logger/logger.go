package logger

import (
	"os"
	"path/filepath"
	"strings"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	log  *zap.Logger
	once sync.Once
)

// Init 初始化 Logger
// mode: "dev" (开发模式，控制台高亮) | "prod" (生产模式，JSON)
// levelStr: "debug", "info", "warn", "error"
func Init(mode string, levelStr string) {
	once.Do(func() {
		var level zapcore.Level
		switch levelStr {
		case "debug":
			level = zapcore.DebugLevel
		case "info":
			level = zapcore.InfoLevel
		case "warn":
			level = zapcore.WarnLevel
		case "error":
			level = zapcore.ErrorLevel
		default:
			level = zapcore.InfoLevel
		}

		var config zap.Config
		if mode == "prod" {
			config = zap.NewProductionConfig()
		} else {
			config = zap.NewDevelopmentConfig()
			config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		}

		config.Level = zap.NewAtomicLevelAt(level)
		// 自定义时间格式
		config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

		var err error
		log, err = config.Build()
		if err != nil {
			panic(err)
		}

		// 记录启动信息
		log.Info("日志系统初始化成功",
			zap.String("mode", mode),
			zap.String("level", level.String()),
		)
	})
}

// AutoInit 自动检测环境并初始化
// - 开发模式（go run）：Console 格式 + Debug 级别
// - 生产模式（编译后/Docker）：JSON 格式 + Info 级别
func AutoInit() {
	mode := detectEnvironment()
	levelStr := detectLogLevel(mode)
	Init(mode, levelStr)
}

// detectEnvironment 检测运行环境
func detectEnvironment() string {
	// 1. 检查环境变量
	if env := os.Getenv("APP_ENV"); env != "" {
		if env == "production" || env == "prod" {
			return "prod"
		}
		return "dev"
	}

	// 2. 检测是否通过 go run 启动
	executable, err := os.Executable()
	if err == nil {
		execName := filepath.Base(executable)
		// go run 会在临时目录生成临时可执行文件
		if strings.Contains(executable, os.TempDir()) ||
			strings.HasPrefix(execName, "go_build_") ||
			strings.HasPrefix(execName, "go-build") {
			return "dev"
		}
	}

	// 3. 检查是否在容器中运行
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return "prod"
	}

	// 4. 默认为生产环境
	return "prod"
}

// detectLogLevel 根据环境检测日志级别
func detectLogLevel(mode string) string {
	// 1. 优先使用环境变量
	if levelStr := os.Getenv("LOG_LEVEL"); levelStr != "" {
		return levelStr
	}

	// 2. 根据环境设置默认级别
	if mode == "dev" {
		return "debug"
	}
	return "info"
}

// Get 返回全局 Logger
func Get() *zap.Logger {
	if log == nil {
		AutoInit() // 默认初始化
	}
	return log
}

// Sync 刷新缓冲
func Sync() {
	if log != nil {
		_ = log.Sync()
	}
}

// Info 快捷方式
func Info(msg string, fields ...zap.Field) {
	Get().Info(msg, fields...)
}

// Error 快捷方式
func Error(msg string, fields ...zap.Field) {
	Get().Error(msg, fields...)
}

// Fatal 快捷方式
func Fatal(msg string, fields ...zap.Field) {
	Get().Fatal(msg, fields...)
}

// Warn 快捷方式
func Warn(msg string, fields ...zap.Field) {
	Get().Warn(msg, fields...)
}

// Debug 快捷方式
func Debug(msg string, fields ...zap.Field) {
	Get().Debug(msg, fields...)
}

// Infow 结构化日志快捷方式（键值对）
func Infow(msg string, keysAndValues ...interface{}) {
	Get().Sugar().Infow(msg, keysAndValues...)
}

// Errorw 结构化日志快捷方式（键值对）
func Errorw(msg string, keysAndValues ...interface{}) {
	Get().Sugar().Errorw(msg, keysAndValues...)
}

// Fatalw 结构化日志快捷方式（键值对）
func Fatalw(msg string, keysAndValues ...interface{}) {
	Get().Sugar().Fatalw(msg, keysAndValues...)
}

// Warnw 结构化日志快捷方式（键值对）
func Warnw(msg string, keysAndValues ...interface{}) {
	Get().Sugar().Warnw(msg, keysAndValues...)
}

// Debugw 结构化日志快捷方式（键值对）
func Debugw(msg string, keysAndValues ...interface{}) {
	Get().Sugar().Debugw(msg, keysAndValues...)
}
