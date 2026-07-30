# Proxy-Cat -- 更稳定的 Mihomo 代理控制塔（五端支持）

Proxy-Cat 是一个基于 Mihomo 的跨平台代理客户端，提供自动稳定节点选择、订阅管理、系统代理控制、代理组视图和连接监控。通过嵌入式 React Web UI 和 Go HTTP API 服务器，在任何支持浏览器的平台上都能运行。

```
                   +--------------------+
                   |    React Web UI    |  浏览器 / WebView
                   |  (frontend/dist/)  |
                   +---------+----------+
                             |
                      HTTP REST API
                      /api/v1/*
                             |
                   +---------+----------+
                   |   Go HTTP Server    |  Go 1.22+
                   |  internal/service/  |
                   +---------+----------+
                   |         |           |
              process   localhost    localhost
              control    HTTP        HTTP
                   |         |           |
          +--------+  +-----+------+  +--+----------+
          | Mihomo |  | System     |  | External    |
          | Core   |  | Proxy      |  | Controller  |
          | (子进程)|  | (OS API)   |  | (Mihomo API)|
          +--------+  +------------+  +-------------+

         +------------------------------------------------+
         |  Thin Shell (按平台自动打开浏览器)              |
         |  Win: rundll32  |  Mac: open  |  Linux: xdg-open |
         |  Android: WebView shell  |  Docker: headless    |
         +------------------------------------------------+
```

## 快速开始

```bash
# 启动服务器（前端自动加载 frontend/dist/）
go run . --port 8080

# 在浏览器中打开
# http://127.0.0.1:8080
```

前置条件：Go 1.22+。无需 Wails CLI、无需 Electron、无需额外桌面运行时。

## 构建

```bash
# 构建所有平台
make build-all

# 单独平台
make build-windows      # proxy-cat-windows-amd64.exe
make build-darwin       # proxy-cat-darwin-amd64 + proxy-cat-darwin-arm64
make build-linux        # proxy-cat-linux-amd64 + proxy-cat-linux-arm64
make build-android      # proxy-cat-android-arm64

# Docker
make build-docker       # docker build -t proxy-cat:latest .

# 清理
make clean

# 运行测试
make test
```

所有二进制文件输出到 `dist/` 目录，包含嵌入式前端资源。

## 目标平台

| 平台      | 模式                | 说明                                      |
|-----------|---------------------|-------------------------------------------|
| Windows   | 桌面（浏览器）      | 自动打开默认浏览器，支持系统代理切换        |
| macOS     | 桌面（浏览器）      | 自动打开默认浏览器，支持系统代理切换        |
| Linux     | 桌面（浏览器）      | 自动打开默认浏览器，支持系统代理切换        |
| Android   | WebView 壳          | 通过 WebView 加载内置 Web UI               |
| Docker    | 无头模式            | 仅 API，不打开浏览器，禁用系统代理功能      |

`cmd/proxy-cat-desktop/` 提供桌面壳层入口：自动寻找空闲端口、启动 API 服务器、调用平台默认浏览器。`main.go` 提供纯 CLI 入口，适合服务器和 Docker 场景。

## 核心功能

- **自动稳定节点选择（Auto-Stable）**：基于延迟采样的自适应评分引擎，自动选择并切换到当前最稳定的代理节点，内置冷却和失败阈值机制，避免频繁切换
- **订阅管理**：通过 URL 加载订阅，自动解析节点、生成 Mihomo 配置文件
- **系统代理开关**：一键启用/禁用操作系统级代理（Windows / macOS / Linux）
- **代理组视图**：可视化展示代理组和节点列表，支持手动切���节点
- **连接监控**：实时查看上传/下载流量、活跃连接数
- **Mihomo 核心管理**：启动/停止/重启 Mihomo 进程，异常退出自动恢复
- **日志查看**：查询最近的应用和核心日志

## 开发

### 前置条件

- Go 1.22+
- Node.js
- pnpm

不再需要 Wails CLI、Electron 或任何桌面框架 SDK。

### 项目结构

```
proxy-cat/
  main.go                          # CLI 入口（--port, --headless, --data-dir）
  cmd/proxy-cat/                   # 原 Wails 入口（已废弃）
  cmd/proxy-cat-desktop/           # 桌面壳层入口（自动打开浏览器）
    main.go                        # 通用启动逻辑
    launcher_windows.go            # Windows 浏览器启动
    launcher_darwin.go             # macOS 浏览器启动
    launcher_linux.go              # Linux 浏览器启动
  internal/
    service/                       # HTTP API 与服务层
      service.go                   # 核心业务逻辑（单写者模式）
      router.go                    # API 路由（Go 1.22 增强路由）
      state.go                     # 状态类型定义
    autostable/                    # 自动稳定引擎
      autostable.go                # 评分、选择、冷却管理
    domain/
      proxy/                       # 代理领域模型
      subscription/                # 订阅解析（Clash / base64）
      configgen/                   # Mihomo YAML 配置生成
    platform/
      mihomo/                      # Mihomo external-controller API 客户端与进程启动器
      system/                      # 系统代理操作（Windows / macOS / Linux）
    group/                         # 代理组健康检查与评分工具
  pkg/api/                         # 共享 API 类型定义（Go + TypeScript 双源）
  frontend/                        # React SPA（Vite + React + Tailwind）
  docs/                            # 架构设计与变更记录
  Makefile                         # 构建脚本
```

### 本地开发

```bash
# 构建前端
cd frontend
pnpm install --frozen-lockfile
pnpm run build          # 输出到 frontend/dist/

# 返回项目根目录，启动服务器
cd ..
go run . --port 8080

# 运行测试
go test ./...
```

### API 概览

| 方法     | 路径                                   | 说明               |
|----------|----------------------------------------|--------------------|
| GET      | `/api/v1/status`                       | 应用状态           |
| GET      | `/api/v1/status/connection`            | 连接统计           |
| POST     | `/api/v1/core/start`                   | 启动 Mihomo 核心   |
| POST     | `/api/v1/core/stop`                    | 停止 Mihomo 核心   |
| POST     | `/api/v1/core/restart`                 | 重启 Mihomo 核心   |
| POST     | `/api/v1/core/recover`                 | 异常恢复           |
| GET      | `/api/v1/system-proxy`                 | 获取系统代理状态   |
| POST     | `/api/v1/system-proxy`                 | 设置系统代理       |
| POST     | `/api/v1/subscription`                 | 加载订阅           |
| GET      | `/api/v1/proxy-groups`                 | 获取代理组列表     |
| PUT      | `/api/v1/proxy-groups/{name}/select`   | 选择代理节点       |
| GET      | `/api/v1/autostable/status`            | 自动稳定状态       |
| PUT      | `/api/v1/autostable/enabled`           | 启停自动稳定       |
| POST     | `/api/v1/autostable/tick`              | 触发一次自动稳定   |
| POST     | `/api/v1/autostable/select`            | 立即选择一个节点   |
| GET      | `/api/v1/logs`                         | 查询日志           |

所有非 API 路径回退到 SPA 入口（`frontend/dist/index.html`），支持前端路由。

## 架构说明

Proxy-Cat 采用 **Go HTTP API + 嵌入式 Web UI** 架构：

- **前端（React）** 通过浏览器的 `fetch` 调用本地 HTTP API，不依赖 Wails bridge 或 IPC 通道。
- **后端（Go）** 以单写者模式管理所有可变状态（`sync.RWMutex` 保护），所有写操作通过 `Service` 方法串行化。
- **Mihomo 进程** 作为独立子进程运行，Proxy-Cat 负责：启动（带参数）、运行时控制（external-controller HTTP API）、异常退出追踪与轻量恢复。
- **系统代理** 通过平台原生 API 操作（Windows 注册表、macOS `networksetup`、Linux `gsettings`），在无头模式下自动禁用。
- **前端资源嵌入** 在构建时将 `frontend/dist/` 目录与二进制文件一起分发，运行时由 Go HTTP 服务器直接提供文件服务。

## 许可

MIT（待定）

