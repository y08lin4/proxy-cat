# Proxy-Cat Phase 0 产品设计冻结

## 1. 产品定义

Proxy-Cat 是一个 Windows 桌面代理客户端，定位为 Clash 增强版客户端，代理内核基于 Mihomo（Clash.Meta）。

它的产品本质是：

- 启动 Mihomo 内核
- 管理节点订阅
- 生成 Mihomo 可运行的 YAML 配置
- 管理规则分流和代理组
- 提供 Windows GUI 控制面板
- 在代理组内提供更稳定的自动选节点能力

Proxy-Cat 不是平台、不是云服务、不是插件系统、不是多租户代理管理系统。它只做一个更聪明、更稳定、更少手动切换节点的 Clash 客户端。

## 2. 阶段边界

### Phase 0：设计冻结

只产出产品和架构设计，不写业务代码。

验收标准：

- 系统边界清晰
- 模块可以直接落到 Wails + Go + Mihomo
- 不引入复杂调度、插件、AI 优化或多级链路
- 后续 Phase 1 到 Phase 4 不需要推翻 Phase 0 设计

### Phase 1：MVP 可运行系统

目标是能启动、能代理、能访问网页、能选择节点。

允许实现：

- Go 项目初始化
- Wails React 基础 UI
- Mihomo 启动器
- YAML 配置生成
- 订阅节点加载
- 基础 proxy group
- 基础 rules
- 系统代理开关

禁止实现：

- 复杂调度系统
- AI 优化
- 多级链路
- 插件系统
- 复杂拓扑或网络可视化

### Phase 2：自动选节点增强

目标是稳定优先的自动切换。

允许实现：

- auto-stable proxy group
- fallback group
- 节点健康检测
- 简单评分缓存
- 抖动控制和冷却

算法边界：

```text
score = latency + failure_rate
```

### Phase 3：UI + 用户体验完善

目标是接近 Clash Verge / FlClash 的基础体验。

允许实现：

- 节点列表
- 代理组
- 当前连接状态
- 开关代理
- 简单日志

禁止实现：

- 复杂拓扑图
- 动画系统
- 实时网络图
- 面向高级运营的平台功能

### Phase 4：稳定性优化 + 收敛

目标是长期稳定运行。

允许实现：

- 自动恢复
- 节点故障切换
- 冷却机制
- 日志系统
- 性能优化

## 3. 简化 IR 模型

Proxy-Cat 内部只保留足够生成 Mihomo 配置和驱动 UI 的中间模型，不建立通用代理编排系统。

核心 IR：

```text
Profile
  - id
  - name
  - subscriptions
  - proxies
  - proxy_groups
  - rules
  - tun/system_proxy settings

ProxyNode
  - id
  - name
  - type
  - server
  - port
  - raw_options

ProxyGroup
  - name
  - type: select | url-test | fallback | auto-stable
  - proxies
  - selected_proxy
  - health

Rule
  - type: domain | domain_suffix | domain_keyword | geoip | match
  - value
  - target_group

HealthSample
  - proxy_id
  - latency_ms
  - success
  - checked_at
```

不做：

- 通用 DAG 链路模型
- 多级代理链编排
- 规则 DSL
- 插件化 IR
- 云端同步数据模型

## 4. 功能范围

Phase 0 冻结后的功能优先级：

1. Mihomo 可启动、可停止、可读取状态
2. 订阅节点能加载到本地 Profile
3. Profile 能生成 Mihomo YAML
4. GUI 能展示节点、代理组、连接状态
5. 用户能开关系统代理并选择节点
6. auto-stable 代理组能用简单评分自动选择更稳定节点

明确非目标：

- 账号系统
- 远程设备管理
- 插件市场
- 脚本规则引擎
- 复杂统计报表
- 多用户权限
- 云配置中心
- 自研代理内核

## 5. Agent A：产品设计结论

产品边界必须始终服务于一句话：

Proxy-Cat 是一个基于 Mihomo 的 Windows Clash 增强客户端，核心差异是稳定优先的自动选节点。

只要某个功能不能直接改善启动、代理、节点选择、规则分流、连接稳定性或基础 GUI 体验，就暂不进入核心范围。

## 6. Agent C：约束审查结论

当前设计没有引入平台化结构。IR 足够表达 Mihomo 配置，但没有抽象成复杂系统。后续可以直接从 Phase 1 开始实现，不需要先建设调度框架或插件框架。
