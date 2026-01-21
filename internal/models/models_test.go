// Package models 数据模型单元测试
package models

import (
	"testing"
	"time"
)

// TestVideoList_SortByNewest 测试视频列表按时间倒序排序
func TestVideoList_SortByNewest(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		input    VideoList
		expected []string // 排序后的 Bvid 顺序
	}{
		{
			name:     "空列表",
			input:    VideoList{},
			expected: []string{},
		},
		{
			name: "单个视频",
			input: VideoList{
				{Bvid: "BV1", TimePosted: now},
			},
			expected: []string{"BV1"},
		},
		{
			name: "多个视频正序输入",
			input: VideoList{
				{Bvid: "BV1", TimePosted: now.Add(-2 * time.Hour)}, // 最旧
				{Bvid: "BV2", TimePosted: now.Add(-1 * time.Hour)}, // 中间
				{Bvid: "BV3", TimePosted: now},                     // 最新
			},
			expected: []string{"BV3", "BV2", "BV1"}, // 按时间倒序
		},
		{
			name: "多个视频乱序输入",
			input: VideoList{
				{Bvid: "BV2", TimePosted: now.Add(-1 * time.Hour)},
				{Bvid: "BV3", TimePosted: now},
				{Bvid: "BV1", TimePosted: now.Add(-2 * time.Hour)},
			},
			expected: []string{"BV3", "BV2", "BV1"},
		},
		{
			name: "相同时间的视频",
			input: VideoList{
				{Bvid: "BV1", TimePosted: now},
				{Bvid: "BV2", TimePosted: now},
			},
			expected: []string{"BV1", "BV2"}, // 相同时间保持原顺序（稳定排序）
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.input.SortByNewest()

			if len(result) != len(tt.expected) {
				t.Errorf("长度不匹配: got %d, want %d", len(result), len(tt.expected))
				return
			}

			for i, bvid := range tt.expected {
				if result[i].Bvid != bvid {
					t.Errorf("位置 %d: got %s, want %s", i, result[i].Bvid, bvid)
				}
			}
		})
	}
}

// TestVideoList_Limit 测试视频列表数量限制
func TestVideoList_Limit(t *testing.T) {
	videos := VideoList{
		{Bvid: "BV1"},
		{Bvid: "BV2"},
		{Bvid: "BV3"},
		{Bvid: "BV4"},
		{Bvid: "BV5"},
	}

	tests := []struct {
		name     string
		input    VideoList
		limit    int
		expected int
	}{
		{
			name:     "限制小于列表长度",
			input:    videos,
			limit:    3,
			expected: 3,
		},
		{
			name:     "限制等于列表长度",
			input:    videos,
			limit:    5,
			expected: 5,
		},
		{
			name:     "限制大于列表长度",
			input:    videos,
			limit:    10,
			expected: 5,
		},
		{
			name:     "限制为 0",
			input:    videos,
			limit:    0,
			expected: 5, // 返回全部
		},
		{
			name:     "限制为负数",
			input:    videos,
			limit:    -1,
			expected: 5, // 返回全部
		},
		{
			name:     "空列表",
			input:    VideoList{},
			limit:    5,
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.input.Limit(tt.limit)

			if len(result) != tt.expected {
				t.Errorf("got %d, want %d", len(result), tt.expected)
			}
		})
	}
}

// TestVideoList_SortByNewest_Limit_Chain 测试链式调用
func TestVideoList_SortByNewest_Limit_Chain(t *testing.T) {
	now := time.Now()
	videos := VideoList{
		{Bvid: "BV3", TimePosted: now.Add(-3 * time.Hour)},
		{Bvid: "BV1", TimePosted: now.Add(-1 * time.Hour)},
		{Bvid: "BV2", TimePosted: now.Add(-2 * time.Hour)},
		{Bvid: "BV4", TimePosted: now.Add(-4 * time.Hour)},
	}

	// 测试 SortByNewest().Limit(2) 应该返回最新的 2 个视频
	result := videos.SortByNewest().Limit(2)

	if len(result) != 2 {
		t.Fatalf("got %d videos, want 2", len(result))
	}

	// 最新的应该是 BV1，其次是 BV2
	if result[0].Bvid != "BV1" {
		t.Errorf("第一个视频应该是 BV1, got %s", result[0].Bvid)
	}
	if result[1].Bvid != "BV2" {
		t.Errorf("第二个视频应该是 BV2, got %s", result[1].Bvid)
	}
}
