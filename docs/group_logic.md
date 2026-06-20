# Proxy-Cat Phase 0 代理组与自动选节点逻辑冻结

## 1. 目标

Proxy-Cat 的核心增强能力是：让代理组自动选择更稳定的节点，减少用户手动切换。

稳定优先不等于最低延迟优先。一个低延迟但频繁失败的节点应排在稳定节点之后。

## 2. 代理组类型

Phase 1：

- `select`：用户手动选择节点
- `url-test`：Mihomo 原生延迟测试组
- `fallback`：Mihomo 原生失败回退组，可选

Phase 2：

- `auto-stable`：Proxy-Cat 管理的稳定优先代理组

## 3. auto-stable 设计边界

`auto-stable` 不是新的代理协议，也不是 Mihomo 扩展。它是 Proxy-Cat 在 Go 后端维护的选择策略。

推荐实现方式：

1. YAML 中生成一个 Mihomo `select` 组，例如 `AUTO-STABLE`
2. Proxy-Cat 定时检测组内节点
3. Go 后端计算分数
4. Go 后端通过 Mihomo external-controller 切换 `AUTO-STABLE` 的选中节点
5. Mihomo 继续负责真实代理连接

这样可以避免改 Mihomo 内核。

## 4. 简单评分规则

基础公式：

```text
score = latency + failure_rate
```

定义：

- `latency`：最近健康检测延迟，单位 ms
- `failure_rate`：最近窗口失败率转换后的惩罚值

建议第一版惩罚：

```text
failure_rate = failed_checks / total_checks * 1000
```

示例：

```text
节点 A：latency = 180ms, failed_checks = 0/10
score = 180 + 0 = 180

节点 B：latency = 80ms, failed_checks = 3/10
score = 80 + 300 = 380
```

节点 A 更稳定，因此优先选择节点 A。

## 5. 健康检测

检测周期：

```text
10s <= interval <= 30s
```

Phase 2 默认值：

```text
interval = 20s
```

检测 URL：

```text
https://www.gstatic.com/generate_204
```

检测结果保存最近窗口：

```text
window_size = 10
```

每个节点缓存：

```text
NodeHealth
  - proxy_id
  - last_latency_ms
  - recent_success_count
  - recent_failure_count
  - last_checked_at
  - score
  - cooldown_until
```

## 6. 缓存规则

必须缓存健康检测结果，避免每次 UI 刷新或每次请求都重新测速。

缓存规则：

- 每个节点维护最近 10 次检测结果
- UI 读取缓存，不触发检测
- YAML 生成不触发检测
- 节点新增后先标记为 unknown，再进入检测队列
- unknown 节点可以临时排在已失败节点之前，但不能优先于稳定低分节点

## 7. 抖动控制

自动选节点必须防止频繁切换。

切换条件：

```text
new_score + switch_threshold < current_score
```

Phase 2 默认：

```text
switch_threshold = 100
min_hold_time = 60s
cooldown_after_failure = 60s
```

含义：

- 新节点必须明显更好才切换
- 当前节点至少保持 60 秒，除非已经不可用
- 刚失败的节点进入冷却，冷却期间不参与优先选择

## 8. fallback 规则

当当前节点连续失败时：

```text
consecutive_failures >= 2
```

允许立即切换到可用分数最低节点，不受 `min_hold_time` 限制。

如果全部节点失败：

- 保持当前选择
- 标记组状态为 degraded
- UI 展示异常
- 下一轮继续检测

## 9. 规则分流关系

规则分流只决定流量进入哪个代理组，不直接选择节点。

示例：

```text
DOMAIN-SUFFIX,google.com,AUTO-STABLE
DOMAIN-SUFFIX,github.com,AUTO-STABLE
GEOIP,CN,DIRECT
MATCH,AUTO-STABLE
```

节点选择只发生在代理组内部。

## 10. dialer-proxy 处理

Phase 1 默认不启用复杂 `dialer-proxy` 链路。

允许边界：

- 如果订阅节点自带 Mihomo 支持的 `dialer-proxy` 字段，YAML 生成器可以原样保留
- Proxy-Cat 不主动生成多级链路
- UI 不提供链路编排

Phase 2 到 Phase 4 仍保持该限制，除非明确进入单独设计阶段。

## 11. Agent A：产品设计结论

用户感知目标是“少手动切节点”。因此 auto-stable 必须简单、可解释、保守切换，而不是追求复杂最优。

## 12. Agent B：架构设计结论

auto-stable 应由 Go 后端计算，Mihomo 执行最终代理。这样既能实现稳定优先，也不会偏离 Mihomo 客户端本质。

## 13. Agent C：约束审查结论

当前算法只依赖延迟、失败率、缓存和冷却，不包含 AI、复杂调度、多级链路或平台化规则。它可以在 Phase 2 独立实现，不会阻塞 Phase 1 MVP。
