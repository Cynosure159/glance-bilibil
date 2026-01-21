# 🎬 glance-bilibil

A [Glance](https://github.com/glanceapp/glance) extension widget to display Bilibili video feeds. Supports multiple UPs, aggregated sorting, and anti-crawling bypass.

[中文文档](./README-ZH.md) · [Quick Start](#-quick-start) · [Glance Integration](#-glance-integration)

---

## ✨ Features

- 👤 **Multi-UP Support**: Monitor multiple Bilibili creators via a single config.
- 🕒 **Chronological Aggregation**: Automatically sorts videos from all configured UPs by post time.
- 🛡️ **Risk Control Bypass**: Implements WBI signing, dynamic `buvid` retrieval, and `dm` parameter simulation for stable access.
- 🎨 **Visual Styles**: Multiple rendering styles (Carousel, Grid, Vertical List).
- ⚙️ **Flexible Config**: Easy configuration via `config.json` with URL parameter overrides.

---

## 🚀 Quick Start

### 1. Configure Creators
Create a `config.json` in the project root:
```json
{
  "port": 8082,
  "channels": [
    { "mid": "946974", "name": "Bilibili Creator A" },
    { "mid": "163637592", "name": "Bilibili Creator B" }
  ],
  "limit": 25,
  "style": "default"
}
```

### 2. Run the Service
Build and start the application:
```bash
go build -o glance-bilibil .
./glance-bilibil -config config.json
```

---

## 🔗 Glance Integration

Add the extension to your `glance.yml`:

```yaml
- type: extension
  url: http://localhost:8082/
  allow-potentially-dangerous-html: true
  cache: 5m
```

### API Endpoints
- `GET /` : Rendered video list (HTML Widget)
- `GET /json` : Aggregated video data (JSON)
- `GET /help` : Configuration help and UP info

---

## 🏗️ Architecture

The project follows a layered design for maintainability:
- **API Layer**: `internal/api/handler.go` - HTTP routing.
- **Service Layer**: `internal/service/video_service.go` - Business logic and concurrency.
- **Platform Layer**: `internal/platform/bilibili.go` - Bilibili API interaction.

---

## 📜 License

MIT License.
