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

## Phase 2 - 自动选节点增强

日期：2026-06-20

变更：

- 新增 `AUTO-STABLE` 默认代理组，内部类型为 `auto-stable`
- Mihomo YAML 中将 `auto-stable` 映射为原生 `select` 组，由 Proxy-Cat 控制选择
- 新增 Mihomo `/proxies/{name}/delay` 延迟检测接口
- 新增 `internal/autostable` 纯策略包
- 实现健康样本缓存，默认窗口 10 次
- 实现冻结评分规则：`score = latency + failure_rate * 1000`
- 实现抖动控制：默认 `switch_threshold = 100`、`min_hold_time = 60s`
- 实现失败冷却：默认 `cooldown_after_failure = 60s`
- 实现连续失败 fallback：默认连续失败 2 次允许绕过保持时间与阈值
- Wails 后端新增 auto-stable 状态、开关、手动 tick 和选择动作
- React 控制面板新增 auto-stable 开关、健康表、手动刷新和 tick 控件

验证：

- `go test -count=1 ./...` 通过
- `tsc --noEmit` 通过
- `vite build` 通过

已知限制：

- 本机仍未检测到全局 `mihomo` 和 `wails` CLI，无法完成真实桌面端代理链路验收
- Phase 2 使用可控 tick，不引入复杂后台调度系统
- 真实健康检测依赖 Mihomo external-controller 运行

验收状态：

- auto-stable 策略、缓存、冷却、fallback 与 UI 已完成代码级闭环
- 真实网络环境下的自动切换需要在提供 `mihomo.exe` 后进行端到端验证

## Phase 3 - UI + 用户体验完善

日期：2026-06-20

变更：

- 新增 Mihomo 连接状态后端接口：核心运行态、上传/下载总量、连接数
- 控制面板新增真实连接状态区，展示连接数、流量、当前 Profile 和 auto-stable 扫描状态
- 代理组视图增加搜索、筛选和更清晰的节点状态展示
- auto-stable 健康区增加摘要：最佳节点、健康数量、平均延迟、失败检查数
- 日志区增加级别筛选、文本过滤和状态色
- UI 标识更新为 Phase 3，并保持 Clash 客户端控制台风格

验证：

- `go test -count=1 ./...` 通过
- `tsc --noEmit` 通过
- `vite build` 通过

已知限制：

- 本机仍未检测到全局 `mihomo` 和 `wails` CLI，无法完成真实 Wails 桌面 smoke test
- Phase 3 未加入复杂拓扑图、动画系统或实时网络图

验收状态：

- 节点列表、代理组、连接状态、系统代理开关、简单日志均已具备
- 真实连接数据依赖 Mihomo external-controller 运行
