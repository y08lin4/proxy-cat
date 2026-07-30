import type { AutoStableNodeHealth } from "../types";

export function formatBytes(value: number): string {
  if (!value) return "0 B";
  const units = ["B", "KB", "MB", "GB", "TB"];
  let i = 0;
  let v = value;
  while (v >= 1024 && i < units.length - 1) { v /= 1024; i++; }
  return `${v.toFixed(i > 0 ? 1 : 0)} ${units[i]}`;
}

export function formatRelativeTime(value?: string): string {
  if (!value) return "--";
  const diff = Date.now() - new Date(value).getTime();
  const seconds = Math.floor(diff / 1000);
  if (seconds < 60) return `${seconds}秒前`;
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes}分钟前`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}小时前`;
  return `${Math.floor(hours / 24)}天前`;
}

export function formatScore(row: AutoStableNodeHealth): string {
  if (row.score === undefined || row.score === null) return "--";
  if (!isFinite(row.score)) return "∞";
  return row.score.toFixed(1);
}

export function formatLatency(row: AutoStableNodeHealth): string {
  if (row.latencyMs === undefined || row.latencyMs === null || row.latencyMs <= 0) return "--";
  return `${row.latencyMs} ms`;
}

export function formatFailureRate(row: AutoStableNodeHealth): string {
  if (row.failureRate === undefined || row.failureRate === null) return "--";
  return `${(row.failureRate * 100).toFixed(1)}%`;
}
