// Unified type definitions for Proxy-Cat frontend
// Single source of truth — mirrors pkg/api/types.go

// ---------------------------------- Request types ----------------------------------

export interface LoadSubscriptionRequest {
  url: string;
  name?: string;
}

export interface SelectProxyRequest {
  proxyName: string;
}

export interface SetSystemProxyRequest {
  enabled: boolean;
}

export interface SetAutoStableEnabledRequest {
  enabled: boolean;
}

export interface SelectAutoStableRequest {
  groupName: string;
}

// ---------------------------------- Response types ----------------------------------

export interface ErrorResponse {
  code: number;
  message: string;
  details?: string;
}

export interface AppStatus {
  coreRunning: boolean;
  systemProxyEnabled: boolean;
  autoStableEnabled: boolean;
  activeProfileName: string;
  controllerAddress: string;
  lastError?: string;
}

export interface ConnectionStatus {
  coreRunning: boolean;
  uploadTotal: number;
  downloadTotal: number;
  connectionCount: number;
}

export interface ProxyGroupView {
  name: string;
  type: string;
  selected: string;
  proxies: ProxyView[];
}

export interface ProxyView {
  name: string;
  type?: string;
  latencyMs?: number;
  alive: boolean;
}

export interface LogLine {
  time: string;
  level: string;
  message: string;
}

export interface AutoStableStatus {
  enabled: boolean;
  available: boolean;
  running: boolean;
  lastTickAt?: string;
  lastAction?: string;
  lastSelected?: string;
  lastError?: string;
  health: AutoStableGroupHealth[];
}

export interface AutoStableGroupHealth {
  name: string;
  type: string;
  selected?: string;
  proxies: AutoStableNodeHealth[];
}

export interface AutoStableNodeHealth {
  name: string;
  type?: string;
  latencyMs?: number;
  alive: boolean;
  score?: number;
  successCount?: number;
  failureCount?: number;
  totalChecks?: number;
  failureRate?: number;
  lastCheckedAt?: string;
  cooldownUntil?: string;
}

export interface AutoStableActionResult {
  action: string;
  groupName?: string;
  selected?: string;
  changed: boolean;
  message?: string;
  completedAt: string;
  health?: AutoStableGroupHealth[];
}

// ---------------------------------- View state ----------------------------------

export type ViewId = "overview" | "proxies" | "auto" | "matrix" | "logs" | "settings";
