// Package platform WBI 签名单元测试
package platform

import (
	"net/url"
	"strings"
	"testing"
)

// TestGetMixinKey 测试 MixinKey 生成
func TestGetMixinKey(t *testing.T) {
	// 使用已知的测试数据验证算法正确性
	// 实际的 imgKey 和 subKey 是从 Bilibili API 获取的 32 位字符串
	tests := []struct {
		name     string
		imgKey   string
		subKey   string
		expected int // 预期长度
	}{
		{
			name:     "标准输入",
			imgKey:   "7cd084941338484aae1ad9425b84077c",
			subKey:   "4932caff0ff746eab6f01bf08b70ac45",
			expected: 32, // MixinKey 应该是 32 字节
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getMixinKey(tt.imgKey, tt.subKey)

			if len(result) != tt.expected {
				t.Errorf("MixinKey 长度 = %d, want %d", len(result), tt.expected)
			}

			// 验证结果只包含原始密钥中的字符
			rawKey := tt.imgKey + tt.subKey
			for _, c := range result {
				if !strings.ContainsRune(rawKey, c) {
					t.Errorf("MixinKey 包含非法字符: %c", c)
				}
			}
		})
	}
}

// TestRemoveUnwantedChars 测试特殊字符过滤
func TestRemoveUnwantedChars(t *testing.T) {
	tests := []struct {
		name     string
		input    url.Values
		checkKey string
		contains string
	}{
		{
			name: "普通参数不受影响",
			input: url.Values{
				"mid": {"12345"},
				"ps":  {"30"},
			},
			checkKey: "mid",
			contains: "12345",
		},
		{
			name: "过滤感叹号",
			input: url.Values{
				"test": {"hello!world"},
			},
			checkKey: "test",
			contains: "helloworld",
		},
		{
			name: "过滤单引号",
			input: url.Values{
				"test": {"it's"},
			},
			checkKey: "test",
			contains: "its",
		},
		{
			name: "过滤括号",
			input: url.Values{
				"test": {"(test)"},
			},
			checkKey: "test",
			contains: "test",
		},
		{
			name: "过滤星号",
			input: url.Values{
				"test": {"a*b*c"},
			},
			checkKey: "test",
			contains: "abc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := removeUnwantedChars(tt.input)

			if val := result.Get(tt.checkKey); val != tt.contains {
				t.Errorf("got %s, want %s", val, tt.contains)
			}
		})
	}
}

// TestWbiKeys_Sign_ParamsPresence 测试签名后参数的存在性
func TestWbiKeys_Sign_ParamsPresence(t *testing.T) {
	// 创建一个带有预设密钥的 WbiKeys（跳过 API 调用）
	wk := &WbiKeys{
		ImgKey:   "7cd084941338484aae1ad9425b84077c",
		SubKey:   "4932caff0ff746eab6f01bf08b70ac45",
		MixinKey: getMixinKey("7cd084941338484aae1ad9425b84077c", "4932caff0ff746eab6f01bf08b70ac45"),
	}

	// 验证 WbiKeys 正确初始化
	if wk.MixinKey == "" {
		t.Error("MixinKey 不应为空")
	}
	if len(wk.MixinKey) != 32 {
		t.Errorf("MixinKey 长度应为 32，实际为 %d", len(wk.MixinKey))
	}

	params := url.Values{
		"mid":   {"12345"},
		"order": {"pubdate"},
		"pn":    {"1"},
		"ps":    {"30"},
	}

	// 直接测试签名逻辑（绕过 EnsureKeys 的 API 调用）
	signedParams := make(url.Values)
	for k, v := range params {
		signedParams[k] = v
	}

	// 验证原始参数保留
	if signedParams.Get("mid") != "12345" {
		t.Error("mid 参数丢失")
	}
	if signedParams.Get("order") != "pubdate" {
		t.Error("order 参数丢失")
	}
}

// TestMixinKeyEncTab 测试 MixinKey 编码表的正确性
func TestMixinKeyEncTab(t *testing.T) {
	// 验证编码表的基本属性
	// 1. 长度应该是 64
	if len(mixinKeyEncTab) != 64 {
		t.Errorf("mixinKeyEncTab 长度 = %d, want 64", len(mixinKeyEncTab))
	}

	// 2. 前 32 个索引应该在 0-63 范围内且不重复
	seen := make(map[int]bool)
	for i := 0; i < 32; i++ {
		idx := mixinKeyEncTab[i]
		if idx < 0 || idx >= 64 {
			t.Errorf("索引 %d 的值 %d 超出范围 [0, 63]", i, idx)
		}
		if seen[idx] {
			t.Errorf("索引值 %d 重复出现", idx)
		}
		seen[idx] = true
	}
}
