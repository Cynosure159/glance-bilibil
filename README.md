<p align="center">
  <h1 align="center">🎬 glance-bilibil</h1>
  <p align="center">
    A <a href="https://github.com/glanceapp/glance">Glance</a> extension widget to display Bilibili video feeds
    <br />
    <a href="./README-ZH.md">中文文档</a> · <a href="#-quick-start">Quick Start</a> · <a href="https://github.com/glanceapp/glance">Glance</a>
  </p>
</p>

<p align="center">
  <a href="https://github.com/Cynosure159/glance-bilibil/actions/workflows/ci.yml">
    <img src="https://github.com/Cynosure159/glance-bilibil/actions/workflows/ci.yml/badge.svg" alt="CI Status" />
  </a>
</p>

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat-square&logo=go" alt="Go Version" />
  <img src="https://img.shields.io/badge/License-MIT-green?style=flat-square" alt="License" />
</p>

## ✨ Features

- 👤 **Multi-UP Support**: Monitor multiple Bilibili creators via a single config.
- 🕒 **Chronological Aggregation**: Automatically sorts videos from all configured UPs by post time.
- 🛡️ **Risk Control Bypass**: Implements WBI signing, dynamic `buvid` retrieval, and `dm` parameter simulation for stable access.
- 🎨 **Visual Styles**: Multiple rendering styles (Carousel, Grid, Vertical List).
- ⚙️ **Flexible Config**: Easy configuration via `config.json` with URL parameter overrides.

## 🚀 Quick Start

### 1. Configure Creators
Create a `config/config.json` in the project root:
```json
{
  "channels": [
    { "mid": "946974", "name": "Bilibili Creator A" },
    { "mid": "163637592", "name": "Bilibili Creator B" }
  ]
}
```

### 2. Run the Service
Build and start the application:
```bash
go build -o glance-bilibil .
./glance-bilibil -config config/config.json -port 8082 -limit 25
```
## 🔗 Glance Integration

Add the extension to your `glance.yml`:

```yaml
- type: extension
  url: http://localhost:8082/
  allow-potentially-dangerous-html: true
  cache: 5m
```

## 📡 API Endpoints
- `GET /` : Rendered video list (HTML Widget)
  - `limit`: Number of videos to display (default: 25).
  - `style`: Visual style: `horizontal-cards` (default), `grid-cards`, `vertical-list`.
  - `mid`: Temporarily filter by a specific UP master MID.
  - `cache`: Cache duration in seconds (default: 300). 0 to disable.
  - `collapse-after`: Collapse vertical list after N items (default: 7).
  - `collapse-after-rows`: Collapse grid after N rows (default: 4).
- `GET /json` : Aggregated video data (JSON)
- `GET /help` : Configuration help and UP info

## 🏗️ Architecture

The project follows a layered design for maintainability:
- **API Layer**: `internal/api/handler.go` - HTTP routing.
- **Service Layer**: `internal/service/video_service.go` - Business logic and concurrency.
- **Platform Layer**: `internal/platform/bilibili.go` - Bilibili API interaction.

## 📜 License

MIT License.
