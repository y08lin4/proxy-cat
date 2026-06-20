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
