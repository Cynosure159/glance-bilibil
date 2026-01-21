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
| Web框架 | Gin (新增) |
| HTTP客户端 | Resty v2 (连接池 + 重试) |
| 模板 | Go Template (embed) |
| 鉴权 | WBI 签名 |
| 并发 | Worker Pool (10 workers) |
| 测试 | Go Testing (单元测试覆盖核心逻辑) |

---

## 🧪 单元测试

### 测试覆盖范围
- **Models**: `VideoList` 的排序与切片逻辑。
- **Worker**: Worker Pool 的并发执行、生命周期管理与错误容错。
- **Platform (WBI)**: 签名算法正确性及特殊字符过滤逻辑。
- **API (Handler)**: 相对时间计算逻辑。

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
| cache | 300 | 缓存时间（秒），0 为禁用 |
| collapse-after | 7 | 垂直列表折叠阈值 |
| collapse-after-rows | 4 | 网格布局折叠行数阈值 |

---

## 🎯 设计模式与优化

### 风控解决方案
为通过 Bilibili 的风控检测，实现了以下机制：
- **buvid cookie**: 从 `/x/frontend/finger/spi` 接口动态获取
- **dm 参数**: 模拟浏览器行为
- **w_webid**: 从用户空间页面解析获取

### HTTP 优化（新增）
**全局单例 Resty 客户端** (`internal/platform/http_client.go`)
- **连接池配置**
  - `MaxIdleConns`: 100 （最大空闲连接）
  - `MaxIdleConnsPerHost`: 10 （每个 Host 最大空闲连接）
  - `IdleConnTimeout`: 90s （空闲连接超时）
- **智能重试策略**
  - 重试次数: 3 次
  - 重试等待: 1s - 5s （指数退避）
  - 重试条件: 网络错误、5xx 错误、429 限流
- **优势**: TCP 连接复用，减少握手开销，提升性能

### 并发处理（优化）
**Worker Pool 模式** (`internal/worker/pool.go`)
- 默认 10 个 Worker，限制并发数量
- 任务队列缓冲：Worker 数量的 2 倍
- **优势**: 防止突发流量耗尽系统资源
- **应用场景**: Service 层获取多个 UP 主视频

**原有并发机制**
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
│   │   ├── wbi.go             # WBI 签名
│   │   └── http_client.go     # HTTP 客户端（新增）
│   ├── service/
│   │   └── video_service.go   # 业务逻辑
│   └── worker/
│       └── pool.go            # Worker Pool（新增）
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
- [x] **HTTP 连接池优化**（2026-01-21）
- [x] **Worker Pool 并发控制**（2026-01-21）
- [x] **智能重试策略**（2026-01-21）
- [x] **核心模块单元测试**（2026-01-21）
  - 实现 Models, Worker, WBI 等模块测试
  - 修复 WBI 特殊字符过滤 Bug

### 待办
- [ ] Docker 支持
- [ ] Glance 集成测试

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
