# Proxy-Cat Changelog

## Phase 0 - 设计冻结

日期：2026-06-20

变更：

- 冻结 Proxy-Cat 产品定义：基于 Mihomo 的 Windows Clash 增强客户端
- 冻结简化 IR：Profile、ProxyNode、ProxyGroup、Rule、HealthSample
- 冻结 Go + Wails + Mihomo 外部进程架构
- 冻结 Mihomo 接入方式：YAML 文件 + localhost external-controller
- 冻结 auto-stable 代理组方向：Go 后端评分，Mihomo 执行选择
- 冻结简单评分规则：`score = latency + failure_rate`
- 明确非目标：平台化、插件系统、AI 优化、多级链路、复杂调度

验收：

- 系统保持 Clash 客户端本质
- Phase 1 可以直接进入 MVP 实现
- Phase 2 自动选节点逻辑有明确实现边界

## Phase 1 - MVP核心可运行系统

日期：2026-06-20

变更：

- 初始化 Go module、Wails 配置、README 和基础忽略规则
- 接入 Mihomo 外部进程控制：启动、停止、重启、状态读取
- 接入 Mihomo localhost external-controller 客户端：读取代理、选择节点、连接管理
- 实现 Windows 系统代理开关的注册表写入骨架
- 实现订阅解析与 Profile 简化 IR：节点、代理组、规则、基础设置
- 实现 Mihomo YAML 生成：端口、controller、proxies、proxy-groups、rules、DNS
- 实现 Phase 1 代理组基础逻辑：`select`、`url-test`、`fallback`
- 保留简单评分函数，为 Phase 2 自动选节点做前置准备，但不启动调度系统
- 实现 Wails 后端绑定：状态、订阅导入、核心启停、系统代理、代理组、节点选择、日志
- 实现 React 基础控制面板：状态卡片、订阅导入、核心控制、系统代理、代理组、节点列表、日志
- 接入 Wails v2 桌面壳和前端资源嵌入入口

验证：

- `go test -count=1 ./...` 通过
- `tsc --noEmit` 通过
- `vite build` 通过

已知限制：

- 当前机器未检测到全局 `mihomo`、`wails`、`pnpm` CLI
- 已完成代码级 MVP 闭环，但真实代理链路仍需要提供 `mihomo.exe` 后进行端到端验证
- Phase 1 不包含 auto-stable 调度、健康检测定时器、冷却机制或复杂 fallback，这些进入 Phase 2

验收状态：

- Go 后端、配置生成、控制面板和 Wails 壳已具备 MVP 形态
- 真实“能访问网页”验收待 Mihomo 二进制和本地运行环境补齐后执行
