# Proxy-Cat Phase 0 架构设计冻结

## 1. 总体架构

Proxy-Cat 使用 Wails 作为 Windows 桌面壳层：

```text
React UI
   |
   | Wails binding
   v
Go backend
   |
   | localhost HTTP / process control / YAML file
   v
Mihomo core
```

职责划分：

- React UI：展示状态、节点、代理组、日志和开关
- Go backend：管理应用状态、订阅、配置生成、Mihomo 进程和系统代理
- Mihomo：执行实际代理、规则匹配、代理组选择、连接管理
- YAML：Proxy-Cat 到 Mihomo 的配置交付边界
- localhost HTTP：Proxy-Cat 到 Mihomo external-controller 的运行时控制边界

## 2. 进程模型

Proxy-Cat 主进程负责启动和守护 Mihomo 子进程。

```text
proxy-cat.exe
  - Wails app
  - Go services
  - React assets

mihomo.exe
  - 由 Proxy-Cat 启动
  - 使用 Proxy-Cat 生成的 config.yaml
  - 暴露 external-controller 到 127.0.0.1
```

Mihomo 只通过本地文件和 localhost HTTP 交互，不引入远程控制面。

## 3. Go 模块划分

建议目录结构：

```text
cmd/proxy-cat/
  main.go

internal/app/
  app.go
  lifecycle.go

internal/core/
  mihomo_launcher.go
  mihomo_api.go
  system_proxy_windows.go

internal/profile/
  profile.go
  store.go
  subscription.go

internal/configgen/
  mihomo_yaml.go
  proxy_group.go
  rules.go

internal/group/
  selector.go
  health.go
  score.go

internal/logs/
  logs.go

frontend/
  src/
```

模块说明：

- `internal/app`：Wails 应用生命周期和前后端绑定入口
- `internal/core`：Mihomo 进程控制、external-controller API、Windows 系统代理
- `internal/profile`：订阅、节点、Profile 本地存储
- `internal/configgen`：把 Proxy-Cat IR 转换为 Mihomo YAML
- `internal/group`：auto-stable 评分、健康检测、抖动控制
- `internal/logs`：应用日志，不替代 Mihomo 日志系统
- `frontend`：React 控制面板

## 4. React UI 模块划分

Phase 1 只做基础控制面板：

```text
frontend/src/
  App.tsx
  api/
    client.ts
  pages/
    Dashboard.tsx
    Proxies.tsx
    Logs.tsx
    Settings.tsx
  components/
    ProxySwitch.tsx
    GroupList.tsx
    NodeList.tsx
    StatusBar.tsx
```

UI 数据从 Go binding 获取。Phase 1 不需要独立前端状态管理框架，使用 React state 和少量 hooks 即可。

## 5. Mihomo 接入方式

Proxy-Cat 不嵌入 Mihomo 库，优先采用外部二进制进程方式：

1. 随应用携带 `mihomo.exe`
2. Go 后端生成 `config.yaml`
3. Go 后端启动 Mihomo：

```text
mihomo.exe -f <app-data>/profiles/active/config.yaml -d <app-data>/mihomo
```

4. 配置中启用 external-controller：

```yaml
external-controller: 127.0.0.1:9090
secret: <local-random-secret>
```

5. Go 后端通过 localhost HTTP 调用 Mihomo API：

```text
GET  /proxies
PUT  /proxies/{group}
GET  /connections
DELETE /connections
GET  /logs
```

所有控制 API 仅绑定 `127.0.0.1`。

## 6. YAML 生成边界

Proxy-Cat 只生成 Mihomo 原生可理解的 YAML，不扩展自定义 runtime。

生成内容：

- `mixed-port`
- `allow-lan`
- `external-controller`
- `secret`
- `proxies`
- `proxy-groups`
- `rules`
- `dns`

Phase 1 默认代理组：

```yaml
proxy-groups:
  - name: PROXY
    type: select
    proxies:
      - AUTO
      - DIRECT
      - <nodes...>

  - name: AUTO
    type: url-test
    url: https://www.gstatic.com/generate_204
    interval: 300
    proxies:
      - <nodes...>
```

Phase 2 增加 Proxy-Cat 管理的 `AUTO-STABLE` 行为。它可以映射为 Mihomo `select` 组，由 Proxy-Cat 根据评分通过 external-controller 选择节点；也可以在保守模式下生成 `fallback` 或 `url-test` 组。首选方案是 Proxy-Cat 计算，Mihomo 执行。

## 7. 数据存储

使用用户本地应用目录：

```text
%APPDATA%/Proxy-Cat/
  profiles/
    active/
      profile.yaml
      config.yaml
  subscriptions/
  mihomo/
  logs/
```

Phase 1 使用本地文件即可，不引入数据库。

## 8. 后端绑定接口

Wails 暴露给前端的最小接口：

```text
GetAppStatus() AppStatus
StartCore() error
StopCore() error
RestartCore() error
SetSystemProxy(enabled bool) error
LoadSubscription(url string) error
GetProxyGroups() []ProxyGroupView
SelectProxy(groupName string, proxyName string) error
GetLogs(limit int) []LogLine
```

Phase 2 增加：

```text
GetHealthStatus() []NodeHealthView
SetAutoStableEnabled(groupName string, enabled bool) error
```

## 9. 错误处理策略

Phase 1：

- Mihomo 启动失败：返回明确错误并展示日志
- YAML 生成失败：阻止启动
- 订阅加载失败：保留旧配置
- 系统代理设置失败：不影响 Mihomo 运行

Phase 4 再完善自动恢复和冷却策略。

## 10. Agent B：架构设计结论

架构以 Mihomo 为唯一代理执行层，Proxy-Cat 只负责配置、进程、UI 和稳定选择策略。这个边界可以避免自研代理逻辑，也能让 Phase 1 快速跑通。

## 11. Agent C：约束审查结论

当前架构没有引入数据库、远程服务、插件系统或复杂规则引擎。Wails binding、YAML、localhost HTTP 三个边界足以支撑 Phase 1 到 Phase 4。
