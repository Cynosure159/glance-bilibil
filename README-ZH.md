# 🎬 glance-bilibil

一个为 [Glance](https://github.com/glanceapp/glance) 开发的 Bilibili 视频汇总展示插件。支持多 UP 主配置、时间轴排序汇总以及完善的风控绕过机制。

[English Document](./README.md) · [快速开始](#-快速开始) · [Glance 集成](#-glance-集成)

---

## ✨ 功能特性

- 👤 **多 UP 主支持**：通过 `config.json` 轻松配置多个感兴趣的 UP 主。
- 🕒 **按时间轴汇总**：自动获取所有配置 UP 主的视频，并按发布时间全局排序。
- 🛡️ **稳定风控绕过**：实现 WBI 签名、动态 `buvid` 获取及 `dm` 参数模拟，绕过 B 站防爬虫机制。
- 🎨 **多种显示样式**：支持轮播 (Default)、网格 (Grid) 和垂直列表 (Vertical List)。
- ⚙️ **配置灵活**：支持配置文件及 URL 参数即时覆盖设置。

---

## 🚀 快速开始

### 1. 准备配置
在项目根目录创建 `config.json`:
```json
{
  "port": 8082,
  "channels": [
    { "mid": "946974", "name": "影视飓风" },
    { "mid": "163637592", "name": "老师好我叫何同学" },
    { "mid": "25876945", "name": "极客湾Geekerwan" }
  ],
  "limit": 25,
  "style": "default"
}
```

### 2. 运行服务
编译并启动：
```bash
go build -o glance-bilibil .
./glance-bilibil -config config.json
```

---

## 🔗 Glance 集成

在你的 `glance.yml` 中添加以下扩展配置：

```yaml
- type: extension
  url: http://localhost:8082/
  allow-potentially-dangerous-html: true
  cache: 5m
```

### API 接口
- `GET /` : 渲染后的视频列表 HTML (供 Glance 嵌入)
- `GET /json` : 聚合后的视频原始数据 (JSON)
- `GET /help` : 使用说明与当前配置详情

---

## 🏗️ 系统架构

本项目采用分层设计以确保可维护性：
- **API 层**: `internal/api/handler.go` - HTTP 路由与处理。
- **服务层**: `internal/service/video_service.go` - 并发汇总与排序算法。
- **平台层**: `internal/platform/bilibili.go` - Bilibili API 客户端与 WBI 签名。

---

## 📜 许可证

MIT License.
