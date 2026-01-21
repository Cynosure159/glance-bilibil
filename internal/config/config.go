// Package config 提供配置加载功能
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config 应用配置
type Config struct {
	Port     int           `json:"port"`     // HTTP 服务端口
	Channels []ChannelInfo `json:"channels"` // UP 主配置列表
	Limit    int           `json:"limit"`    // 默认显示视频数量
	Style    string        `json:"style"`    // 默认显示样式
}

// ChannelInfo UP 主信息
type ChannelInfo struct {
	Mid  string `json:"mid"`  // 用户 UID
	Name string `json:"name"` // 名称（可选，用于显示）
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		Port:     8082,
		Channels: []ChannelInfo{},
		Limit:    25,
		Style:    "default",
	}
}

// Load 从文件加载配置
func Load(path string) (*Config, error) {
	// 如果路径为空，尝试默认路径
	if path == "" {
		path = getDefaultConfigPath()
	}

	// 检查文件是否存在
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return DefaultConfig(), nil
	}

	// 读取文件
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	// 解析 JSON
	cfg := DefaultConfig()
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	// 验证配置
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate 验证配置
func (c *Config) Validate() error {
	if c.Port <= 0 || c.Port > 65535 {
		c.Port = 8080
	}
	if c.Limit <= 0 {
		c.Limit = 25
	}
	if c.Style == "" {
		c.Style = "default"
	}
	return nil
}

// getDefaultConfigPath 获取默认配置文件路径
func getDefaultConfigPath() string {
	// 1. 优先使用环境变量
	if path := os.Getenv("CONFIG_PATH"); path != "" {
		return path
	}

	// 2. 检查当前工作目录
	defaultName := filepath.Join("config", "config.json")
	if _, err := os.Stat(defaultName); err == nil {
		return defaultName
	}

	// 3. 检查可执行文件所在目录
	exe, err := os.Executable()
	if err == nil {
		path := filepath.Join(filepath.Dir(exe), "config", "config.json")
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	// 4. 回退到默认值
	return defaultName
}

// Save 保存配置到文件
func (c *Config) Save(path string) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化配置失败: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("写入配置文件失败: %w", err)
	}

	return nil
}
