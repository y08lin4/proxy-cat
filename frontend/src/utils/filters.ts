import type { AutoStableGroupHealth, AutoStableNodeHealth, LogLine, ProxyGroupView } from "../types";

export interface HealthRow extends AutoStableNodeHealth {
  groupName: string;
  groupType: string;
}

export function flattenHealth(groups: AutoStableGroupHealth[]): HealthRow[] {
  if (!groups) return [];
  return groups.flatMap(g => g.proxies.map(p => ({ groupName: g.name, groupType: g.type, ...p })));
}

export interface HealthSummary {
  total: number;
  healthy: number;
  failures: number;
  bestNode: string;
  averageLatency: number;
}

export function summarizeHealth(rows: HealthRow[]): HealthSummary {
  const total = rows.length;
  const healthy = rows.filter(r => r.alive).length;
  const failures = rows.filter(r => r.failureCount && r.failureCount > 0).length;
  const aliveRows = rows.filter(r => r.alive && r.latencyMs && r.latencyMs > 0);
  const averageLatency = aliveRows.length > 0
    ? Math.round(aliveRows.reduce((s, r) => s + (r.latencyMs ?? 0), 0) / aliveRows.length)
    : 0;
  const sorted = [...rows].sort((a, b) => (a.score ?? Infinity) - (b.score ?? Infinity));
  const bestNode = sorted.length > 0 ? sorted[0].name : "--";
  return { total, healthy, failures, bestNode, averageLatency };
}

export function healthScore(row: AutoStableNodeHealth): number {
  return row.score ?? Infinity;
}

export function filterGroups(groups: ProxyGroupView[], query: string, view: string): ProxyGroupView[] {
  if (!groups) return [];
  let filtered = groups;
  if (view === "selected") filtered = filtered.filter(g => g.selected && g.selected !== "");
  if (view === "auto") filtered = filtered.filter(g => g.type === "auto-stable" || g.name.toLowerCase().includes("stable"));
  if (query) {
    const q = query.toLowerCase();
    filtered = filtered.filter(g => g.name.toLowerCase().includes(q) || g.type.toLowerCase().includes(q));
  }
  return filtered;
}

export function filterLogs(logs: LogLine[], level: string, query: string): LogLine[] {
  if (!logs) return [];
  let filtered = logs;
  if (level && level !== "all") filtered = filtered.filter(l => l.level === level);
  if (query) {
    const q = query.toLowerCase();
    filtered = filtered.filter(l => l.message.toLowerCase().includes(q));
  }
  return filtered;
}

export function uniqueLogLevels(logs: LogLine[]): string[] {
  if (!logs) return [];
  return [...new Set(logs.map(l => l.level))];
}
