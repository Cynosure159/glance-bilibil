// Package platform 提供 WBI 签名功能
package platform

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

// MIXIN_KEY_ENC_TAB 是 WBI 签名算法中使用的字符重排映射表
var mixinKeyEncTab = []int{
	46, 47, 18, 2, 53, 8, 23, 32, 15, 50, 10, 31, 58, 3, 45, 35, 27, 43, 5, 49,
	33, 9, 42, 19, 29, 28, 14, 39, 12, 38, 41, 13, 37, 48, 7, 16, 24, 55, 40,
	61, 26, 17, 0, 1, 60, 51, 30, 4, 22, 25, 54, 21, 56, 59, 6, 63, 57, 62, 11,
	36, 20, 34, 44, 52,
}

// WbiKeys 存储 WBI 签名所需的密钥信息
type WbiKeys struct {
	ImgKey         string
	SubKey         string
	MixinKey       string
	LastUpdateTime time.Time
	mu             sync.RWMutex
}

// 全局 WBI 密钥实例
var wbiKeys = &WbiKeys{}

// navResponse 用于解析 Bilibili nav 接口返回的 JSON
type navResponse struct {
	Code int `json:"code"`
	Data struct {
		WbiImg struct {
			ImgUrl string `json:"img_url"`
			SubUrl string `json:"sub_url"`
		} `json:"wbi_img"`
	} `json:"data"`
}

// getMixinKey 通过 MIXIN_KEY_ENC_TAB 重排 imgKey 和 subKey，生成 mixinKey
func getMixinKey(imgKey, subKey string) string {
	rawWbiKey := imgKey + subKey
	var mixinKey strings.Builder
	for i := 0; i < 32; i++ {
		mixinKey.WriteByte(rawWbiKey[mixinKeyEncTab[i]])
	}
	return mixinKey.String()
}

// Update 从 Bilibili 获取最新的 WBI 密钥
func (wk *WbiKeys) Update() error {
	wk.mu.Lock()
	defer wk.mu.Unlock()

	// 使用全局 Resty 客户端
	client := GetRestyClient()

	var nav navResponse
	resp, err := client.R().
		SetHeader("Referer", "https://www.bilibili.com/").
		SetResult(&nav).
		Get("https://api.bilibili.com/x/web-interface/nav")

	if err != nil {
		return fmt.Errorf("请求失败: %w", err)
	}

	if !resp.IsSuccess() {
		return fmt.Errorf("HTTP 错误: %d", resp.StatusCode())
	}

	if nav.Code != 0 && nav.Code != -101 {
		return fmt.Errorf("API 返回错误码: %d", nav.Code)
	}

	imgUrl := nav.Data.WbiImg.ImgUrl
	subUrl := nav.Data.WbiImg.SubUrl
	if imgUrl == "" || subUrl == "" {
		return fmt.Errorf("获取密钥失败: img_url 或 sub_url 为空")
	}

	imgParts := strings.Split(imgUrl, "/")
	subParts := strings.Split(subUrl, "/")
	wk.ImgKey = strings.TrimSuffix(imgParts[len(imgParts)-1], ".png")
	wk.SubKey = strings.TrimSuffix(subParts[len(subParts)-1], ".png")
	wk.MixinKey = getMixinKey(wk.ImgKey, wk.SubKey)
	wk.LastUpdateTime = time.Now()

	return nil
}

// EnsureKeys 确保密钥有效
func (wk *WbiKeys) EnsureKeys() error {
	wk.mu.RLock()
	needUpdate := time.Since(wk.LastUpdateTime) > time.Hour || wk.MixinKey == ""
	wk.mu.RUnlock()

	if needUpdate {
		return wk.Update()
	}
	return nil
}

// Sign 对请求参数进行 WBI 签名
func (wk *WbiKeys) Sign(params url.Values) (url.Values, error) {
	if err := wk.EnsureKeys(); err != nil {
		return nil, fmt.Errorf("获取 WBI 密钥失败: %w", err)
	}

	wk.mu.RLock()
	mixinKey := wk.MixinKey
	wk.mu.RUnlock()

	signedParams := make(url.Values)
	for k, v := range params {
		signedParams[k] = v
	}
	signedParams.Set("wts", strconv.FormatInt(time.Now().Unix(), 10))
	signedParams = removeUnwantedChars(signedParams)

	query := signedParams.Encode()
	hash := md5.Sum([]byte(query + mixinKey))
	wRid := hex.EncodeToString(hash[:])
	signedParams.Set("w_rid", wRid)

	return signedParams, nil
}

// removeUnwantedChars 过滤掉 WBI 签名不允许的特殊字符
func removeUnwantedChars(v url.Values) url.Values {
	encoded := v.Encode()
	chars := []byte{'!', '\'', '(', ')', '*'}
	b := []byte(encoded)
	for _, c := range chars {
		b = bytes.ReplaceAll(b, []byte{c}, nil)
	}
	result, err := url.ParseQuery(string(b))
	if err != nil {
		return v
	}
	return result
}

// GetWbiKeys 获取全局 WBI 密钥实例
func GetWbiKeys() *WbiKeys {
	return wbiKeys
}
