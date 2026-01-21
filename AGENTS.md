# glance-bilibil 项目文档

> 项目知识库 - 为 AI Agents 和开发者提供完整项目上下文

## 📋 项目概述

glance-bilibil 是一个为 [glance](https://github.com/glanceapp/glance) 开发的 Bilibili 视频扩展插件。支持配置多个 UP 主，按时间排序汇总显示最新视频。

## 🏗️ 系统架构

### 分层设计
```
Glance 看板
    ↓ HTTP GET /videos
glance-bilibil (HTTP Server)
    ├── API 层 (handler.go) - HTTP 路由
    ├── 服务层 (video_service.go) - 并发获取、排序汇总
    ├── 平台层 (bilibili.go) - Bilibili API 客户端
    │   └── wbi.go - WBI 签名
    ├── 配置层 (config.go) - 配置加载
    └── 模型层 (models.go) - 数据结构
    ↓ REST API
Bilibili API
```

### 核心数据模型
```go
// 视频信息 - internal/models/models.go
type Video struct {
    Title        string    // 视频标题
    ThumbnailUrl string    // 缩略图
    Url          string    // 视频链接
    Author       string    // UP 主名称
    AuthorUrl    string    // UP 主主页
    TimePosted   time.Time // 发布时间
    Duration     string    // 时长
    PlayCount    int       // 播放量
    Bvid         string    // BV 号
}
```

---

## ⚙️ 配置与部署

### 配置文件 (`config/config.json`)
```json
{
  "channels": [
    { "mid": "946974", "name": "影视飓风" },
    { "mid": "163637592", "name": "老师好我叫何同学" },
    { "mid": "25876945", "name": "极客湾Geekerwan" }
  ]
}
```

**命令行参数**：
- `-port`: HTTP 服务端口（默认 `8082`）
- `-limit`: 默认显示视频数量（默认 `25`）
- `-config`: 配置文件路径（默认 `./config/config.json`）

**环境变量**：
- `CONFIG_PATH`: 配置文件路径（默认 `./config/config.json`）

### Glance 集成
```yaml
# glance.yml
- type: extension
  url: http://localhost:8082/
  allow-potentially-dangerous-html: true
  cache: 5m
```

---

## 🔧 技术栈

| 组件 | 技术 |
|------|------|
| 语言 | Go 1.21+ |
| HTTP | 标准库 net/http |
| 模板 | Go Template (embed) |
| 鉴权 | WBI 签名 |

---

## 📡 API 端点

| 端点 | 方法 | 描述 |
|------|------|------|
| `/` | GET | HTML Widget（供 Glance 嵌入）|
| `/json` | GET | JSON 格式的排序视频列表 |
| `/help` | GET | 帮助/说明页 |
| `/health` | GET | 健康检查 |

**参数**：
| 参数 | 默认值 | 说明 |
|------|--------|------|
| limit | 25 (flag) | 显示视频数量 |
| style | horizontal-cards | 样式: horizontal-cards/grid-cards/vertical-list |
| mid | - | 临时指定单个 UP 主 |
| collapse-after | 7 | 垂直列表折叠阈值 |
| collapse-after-rows | 4 | 网格布局折叠行数阈值 |

---

## 🎯 设计模式与优化

### 风控解决方案
为通过 Bilibili 的风控检测，实现了以下机制：
- **buvid cookie**: 从 `/x/frontend/finger/spi` 接口动态获取
- **dm 参数**: 模拟浏览器行为
- **w_webid**: 从用户空间页面解析获取

### 并发处理
- 使用 goroutine 并发获取多个 UP 主的视频
- WaitGroup 同步，channel 收集结果
- 单个失败不影响其他 UP 主

---

## 📁 目录结构
```
glance-bilibil/
├── main.go                    # 入口
├── config/                    # 配置目录
│   └── config.json            # 配置文件
├── go.mod                     # Go 模块
├── internal/
│   ├── api/handler.go         # HTTP 处理器
│   ├── config/config.go       # 配置加载
│   ├── models/models.go       # 数据结构
│   ├── platform/
│   │   ├── bilibili.go        # Bilibili 客户端
│   │   └── wbi.go             # WBI 签名
│   └── service/
│       └── video_service.go   # 业务逻辑
├── templates/                 # HTML 模板
│   ├── videos.html
│   ├── videos-grid.html
│   ├── videos-list.html
│   └── video-card.html
└── doc/                       # 参考文档
```

---

## 📊 开发状态

### 已完成
- [x] 分层架构设计
- [x] 多 UP 主配置支持
- [x] 并发获取与时间排序汇总
- [x] WBI 签名与风控绕过
- [x] 三种展示样式
- [x] 配置文件支持

### 待办
- [ ] Docker 支持
- [ ] Glance 集成测试
- [ ] 缓存机制

---

## 🔮 开发规范

### 代码规范
- 所有公开函数必须添加注释
- 错误处理使用 Go 标准错误模式
- 日志使用 `log.Printf` 并带等级标签

### 添加新功能
1. 模型层：在 `internal/models/` 添加数据结构
2. 平台层：在 `internal/platform/` 添加 API 客户端
3. 服务层：在 `internal/service/` 添加业务逻辑
4. API 层：在 `internal/api/` 添加处理器
