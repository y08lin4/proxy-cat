export function translateGroupType(type: string): string {
  switch (type) {
    case "select": return "手动选择";
    case "url-test": return "自动测速";
    case "fallback": return "故障转移";
    case "auto-stable": return "自动稳定";
    case "load-balance": return "负载均衡";
    case "relay": return "链式代理";
    default: return type || "未知";
  }
}

export function translateLogLevel(level: string): string {
  switch (level) {
    case "fatal": return "致命";
    case "error": return "错误";
    case "warn": return "警告";
    case "info": return "信息";
    case "debug": return "调试";
    case "trace": return "追踪";
    default: return level;
  }
}

export function proxyMeta(proxy: { type?: string; latencyMs?: number; alive: boolean }): string {
  if (!proxy.alive) return "离线";
  if (proxy.latencyMs !== undefined && proxy.latencyMs > 0) return `${proxy.latencyMs}ms`;
  return proxy.type || "";
}

export function logTone(level: string): "bad" | "warn" | "faded" | "info" {
  switch (level) {
    case "error":
    case "fatal": return "bad";
    case "warn": return "warn";
    case "debug":
    case "trace": return "faded";
    default: return "info";
  }
}
