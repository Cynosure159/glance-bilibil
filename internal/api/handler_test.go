// Package api HTTP 处理器单元测试
package api

import (
	"testing"
	"time"
)

// TestRelativeTime 测试相对时间计算
func TestRelativeTime(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		input    time.Time
		expected string
	}{
		{
			name:     "刚刚（30秒前）",
			input:    now.Add(-30 * time.Second),
			expected: "刚刚",
		},
		{
			name:     "几分钟前",
			input:    now.Add(-5 * time.Minute),
			expected: "5m",
		},
		{
			name:     "几小时前",
			input:    now.Add(-3 * time.Hour),
			expected: "3h",
		},
		{
			name:     "几天前",
			input:    now.Add(-5 * 24 * time.Hour),
			expected: "5d",
		},
		{
			name:     "几个月前",
			input:    now.Add(-60 * 24 * time.Hour), // 约 2 个月
			expected: "2mo",
		},
		{
			name:     "几年前",
			input:    now.Add(-400 * 24 * time.Hour), // 约 1 年
			expected: "1y",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := relativeTime(tt.input)

			if result != tt.expected {
				t.Errorf("relativeTime() = %s, want %s", result, tt.expected)
			}
		})
	}
}

// TestRelativeTime_EdgeCases 测试边界情况
func TestRelativeTime_EdgeCases(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{
			name:     "恰好 1 分钟",
			duration: -1 * time.Minute,
			expected: "1m",
		},
		{
			name:     "恰好 1 小时",
			duration: -1 * time.Hour,
			expected: "1h",
		},
		{
			name:     "恰好 1 天",
			duration: -24 * time.Hour,
			expected: "1d",
		},
		{
			name:     "恰好 30 天",
			duration: -30 * 24 * time.Hour,
			expected: "1mo",
		},
		{
			name:     "恰好 365 天",
			duration: -365 * 24 * time.Hour,
			expected: "1y",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := relativeTime(now.Add(tt.duration))

			if result != tt.expected {
				t.Errorf("relativeTime() = %s, want %s", result, tt.expected)
			}
		})
	}
}

// TestDefaultStyle 测试默认样式常量
func TestDefaultStyle(t *testing.T) {
	if DefaultStyle != "horizontal-cards" {
		t.Errorf("DefaultStyle = %s, want horizontal-cards", DefaultStyle)
	}
}
