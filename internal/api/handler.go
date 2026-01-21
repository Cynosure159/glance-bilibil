// Package api 提供 HTTP 处理器
package api

import (
	"embed"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"time"

	"encoding/json"
	"glance-bilibil/internal/config"
	"glance-bilibil/internal/models"
	"glance-bilibil/internal/service"
)

// HelpData 帮助页面模板数据
type HelpData struct {
	Channels     []config.ChannelInfo
	DefaultLimit int
	DefaultStyle string
}

// Handler HTTP 处理器
type Handler struct {
	service      *service.VideoService
	templates    map[string]*template.Template
	defaultLimit int
	defaultStyle string
}

// TemplateData 传递给模板的数据
type TemplateData struct {
	Videos            models.VideoList
	Style             string
	CollapseAfter     int
	CollapseAfterRows int
}

const (
	DefaultStyle = "horizontal-cards"
)

// NewHandler 创建处理器
func NewHandler(svc *service.VideoService, templatesFS embed.FS, defaultLimit int) (*Handler, error) {
	h := &Handler{
		service:      svc,
		templates:    make(map[string]*template.Template),
		defaultLimit: defaultLimit,
		defaultStyle: DefaultStyle,
	}

	funcMap := template.FuncMap{
		"relativeTime": relativeTime,
		"safeURL":      func(s string) template.URL { return template.URL(s) },
	}

	// 加载模板
	var err error

	h.templates["horizontal-cards"], err = template.New("videos.html").Funcs(funcMap).ParseFS(
		templatesFS, "templates/videos.html", "templates/video-card.html")
	if err != nil {
		return nil, err
	}

	h.templates["grid-cards"], err = template.New("videos-grid.html").Funcs(funcMap).ParseFS(
		templatesFS, "templates/videos-grid.html", "templates/video-card.html")
	if err != nil {
		return nil, err
	}

	h.templates["vertical-list"], err = template.New("videos-list.html").Funcs(funcMap).ParseFS(
		templatesFS, "templates/videos-list.html")
	if err != nil {
		return nil, err
	}

	h.templates["help"], err = template.New("help.html").Funcs(funcMap).ParseFS(
		templatesFS, "templates/help.html")
	if err != nil {
		return nil, err
	}

	return h, nil
}

// relativeTime 计算相对时间
func relativeTime(t time.Time) string {
	duration := time.Since(t)
	switch {
	case duration < time.Minute:
		return "刚刚"
	case duration < time.Hour:
		return strconv.Itoa(int(duration.Minutes())) + "m"
	case duration < 24*time.Hour:
		return strconv.Itoa(int(duration.Hours())) + "h"
	case duration < 30*24*time.Hour:
		return strconv.Itoa(int(duration.Hours()/24)) + "d"
	case duration < 365*24*time.Hour:
		return strconv.Itoa(int(duration.Hours()/24/30)) + "mo"
	default:
		return strconv.Itoa(int(duration.Hours()/24/365)) + "y"
	}
}

// VideosHandler 处理视频列表请求
func (h *Handler) VideosHandler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	// 解析参数（URL 参数可覆盖默认值）
	limit := h.defaultLimit
	if limitStr := query.Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	style := h.defaultStyle
	if s := query.Get("style"); s != "" {
		style = s
	}

	collapseAfter := 7
	if ca := query.Get("collapse-after"); ca != "" {
		if v, err := strconv.Atoi(ca); err == nil && v > 0 {
			collapseAfter = v
		}
	}

	collapseAfterRows := 4
	if car := query.Get("collapse-after-rows"); car != "" {
		if v, err := strconv.Atoi(car); err == nil && v > 0 {
			collapseAfterRows = v
		}
	}

	cacheTTL := 300 // 默认 5 分钟
	if cStr := query.Get("cache"); cStr != "" {
		if c, err := strconv.Atoi(cStr); err == nil && c >= 0 {
			cacheTTL = c
		}
	}

	// 检查是否有临时指定的单个 mid
	var videos models.VideoList
	var err error

	if mid := query.Get("mid"); mid != "" {
		// 单个 UP 主模式
		videos, err = h.service.FetchChannelVideos(mid, limit, cacheTTL)
	} else {
		// 多 UP 主汇总模式
		videos, err = h.service.FetchAllVideos(limit, cacheTTL)
	}

	if err != nil {
		log.Printf("[ERROR] 获取视频失败: %v", err)
		http.Error(w, "获取视频失败: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 准备模板数据
	data := TemplateData{
		Videos:            videos,
		Style:             style,
		CollapseAfter:     collapseAfter,
		CollapseAfterRows: collapseAfterRows,
	}

	// 选择模板
	tmpl, ok := h.templates[style]
	if !ok {
		tmpl = h.templates["horizontal-cards"]
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Widget-Title", "Bilibili")
	w.Header().Set("Widget-Title-URL", "https://www.bilibili.com")
	w.Header().Set("Widget-Content-Type", "html")
	frameless := "true"
	if style == "vertical-list" {
		frameless = "false"
	}
	w.Header().Set("Widget-Content-Frameless", frameless)
	if err := tmpl.Execute(w, data); err != nil {
		log.Printf("[ERROR] 渲染模板失败: %v", err)
		http.Error(w, "渲染失败", http.StatusInternalServerError)
	}
}

// JSONHandler 以 JSON 格式输出排序后的视频列表
func (h *Handler) JSONHandler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	limit := h.defaultLimit
	if limitStr := query.Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	cacheTTL := 300
	if cStr := query.Get("cache"); cStr != "" {
		if c, err := strconv.Atoi(cStr); err == nil && c >= 0 {
			cacheTTL = c
		}
	}

	var videos models.VideoList
	var err error

	if mid := query.Get("mid"); mid != "" {
		videos, err = h.service.FetchChannelVideos(mid, limit, cacheTTL)
	} else {
		videos, err = h.service.FetchAllVideos(limit, cacheTTL)
	}

	if err != nil {
		log.Printf("[ERROR] 获取视频失败: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(videos)
}

// HealthHandler 健康检查
func (h *Handler) HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// HelpHandler 帮助说明页
func (h *Handler) HelpHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/help" {
		http.NotFound(w, r)
		return
	}

	cfg := h.service.GetConfig()
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	data := HelpData{
		Channels:     cfg.Channels,
		DefaultLimit: h.defaultLimit,
		DefaultStyle: h.defaultStyle,
	}

	if err := h.templates["help"].Execute(w, data); err != nil {
		log.Printf("[ERROR] 渲染帮助页面失败: %v", err)
		http.Error(w, "渲染失败", http.StatusInternalServerError)
	}
}
