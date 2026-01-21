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
	"glance-bilibil/internal/models"
	"glance-bilibil/internal/service"
)

// Handler HTTP 处理器
type Handler struct {
	service   *service.VideoService
	templates map[string]*template.Template
}

// TemplateData 传递给模板的数据
type TemplateData struct {
	Videos            models.VideoList
	Style             string
	CollapseAfter     int
	CollapseAfterRows int
}

// NewHandler 创建处理器
func NewHandler(svc *service.VideoService, templatesFS embed.FS) (*Handler, error) {
	h := &Handler{
		service:   svc,
		templates: make(map[string]*template.Template),
	}

	funcMap := template.FuncMap{
		"relativeTime": relativeTime,
		"safeURL":      func(s string) template.URL { return template.URL(s) },
	}

	// 加载模板
	var err error

	h.templates["default"], err = template.New("videos.html").Funcs(funcMap).ParseFS(
		templatesFS, "templates/videos.html", "templates/video-card.html")
	if err != nil {
		return nil, err
	}

	h.templates["grid"], err = template.New("videos-grid.html").Funcs(funcMap).ParseFS(
		templatesFS, "templates/videos-grid.html", "templates/video-card.html")
	if err != nil {
		return nil, err
	}

	h.templates["vertical-list"], err = template.New("videos-list.html").Funcs(funcMap).ParseFS(
		templatesFS, "templates/videos-list.html")
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
	cfg := h.service.GetConfig()

	// 解析参数（URL 参数可覆盖配置）
	limit := cfg.Limit
	if limitStr := query.Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	style := cfg.Style
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

	// 检查是否有临时指定的单个 mid
	var videos models.VideoList
	var err error

	if mid := query.Get("mid"); mid != "" {
		// 单个 UP 主模式
		videos, err = h.service.FetchChannelVideos(mid, limit)
	} else {
		// 多 UP 主汇总模式
		videos, err = h.service.FetchAllVideos(limit)
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
		tmpl = h.templates["default"]
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Widget-Title", "Bilibili")
	w.Header().Set("Widget-Title-URL", "https://www.bilibili.com")
	w.Header().Set("Widget-Content-Type", "html")
	w.Header().Set("Widget-Content-Frameless", "true")
	if err := tmpl.Execute(w, data); err != nil {
		log.Printf("[ERROR] 渲染模板失败: %v", err)
		http.Error(w, "渲染失败", http.StatusInternalServerError)
	}
}

// JSONHandler 以 JSON 格式输出排序后的视频列表
func (h *Handler) JSONHandler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	cfg := h.service.GetConfig()

	limit := cfg.Limit
	if limitStr := query.Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	var videos models.VideoList
	var err error

	if mid := query.Get("mid"); mid != "" {
		videos, err = h.service.FetchChannelVideos(mid, limit)
	} else {
		videos, err = h.service.FetchAllVideos(limit)
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

	// 构建 UP 主列表
	channelList := ""
	for _, ch := range cfg.Channels {
		channelList += "<li>" + ch.Name + " (mid: " + ch.Mid + ")</li>"
	}
	if channelList == "" {
		channelList = "<li>未配置 UP 主</li>"
	}

	html := `<!DOCTYPE html>
<html>
<head>
	<title>glance-bilibil</title>
	<style>
		body { font-family: system-ui, sans-serif; max-width: 800px; margin: 50px auto; padding: 0 20px; }
		h1 { color: #00a1d6; }
		code { background: #f4f4f4; padding: 2px 6px; border-radius: 3px; }
		table { border-collapse: collapse; width: 100%%; margin: 20px 0; }
		th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
		th { background: #f8f8f8; }
	</style>
</head>
<body>
	<h1>🎬 glance-bilibil</h1>
	<p>为 <a href="https://github.com/glanceapp/glance">glance</a> 开发的 Bilibili 视频扩展插件。</p>
	
	<h2>已配置的 UP 主</h2>
	<ul>` + channelList + `</ul>
	
	<h2>使用方法</h2>
	<p><code>GET /</code> - 获取所有 UP 主的视频汇总 HTML（供 Glance 嵌入）</p>
	<p><code>GET /json</code> - 获取所有 UP 主的视频汇总 JSON</p>
	<p><code>GET /help</code> - 本帮助说明页</p>

	<h2>参数说明</h2>
	<table>
		<tr><th>参数</th><th>默认值</th><th>说明</th></tr>
		<tr><td>limit</td><td>` + strconv.Itoa(cfg.Limit) + `</td><td>显示视频数量</td></tr>
		<tr><td>style</td><td>` + cfg.Style + `</td><td>样式: default/grid/vertical-list</td></tr>
		<tr><td>mid</td><td>-</td><td>临时指定单个 UP 主 UID</td></tr>
	</table>
	
	<h2>示例</h2>
	<ul>
		<li><a href="/">/</a> - 所有 UP 主视频汇总</li>
		<li><a href="/json">/json</a> - 获取 JSON 格式视频汇总</li>
		<li><a href="/?limit=10&style=grid">/?limit=10&style=grid</a></li>
		<li><a href="/?mid=946974&limit=5">/?mid=946974&limit=5</a> - 单个 UP</li>
	</ul>
</body>
</html>`
	w.Write([]byte(html))
}
