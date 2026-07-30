// Proxy-Cat API client — communicates with the Go HTTP API server
import type {
  AppStatus,
  AutoStableActionResult,
  AutoStableStatus,
  ConnectionStatus,
  LogLine,
  ProfileMeta,
  ProxyGroupView,
  Rule,
  RuleTemplate,
  RuleValidationError,
} from "../types";

const BASE_URL = "";
const DEFAULT_TIMEOUT = 30_000;

class ApiError extends Error {
  constructor(
    public code: number,
    message: string,
  ) {
    super(message);
    this.name = "ApiError";
  }
}

async function request<T>(
  method: string,
  path: string,
  body?: unknown,
  timeout = DEFAULT_TIMEOUT,
): Promise<T> {
  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), timeout);
  try {
    const response = await fetch(`${BASE_URL}${path}`, {
      method,
      headers: body ? { "Content-Type": "application/json" } : undefined,
      body: body ? JSON.stringify(body) : undefined,
      signal: controller.signal,
    });
    if (!response.ok) {
      let message = response.statusText;
      try {
        const err = await response.json();
        message = err.message || message;
      } catch { /* use status text */ }
      throw new ApiError(response.status, message);
    }
    if (response.status === 204) return undefined as T;
    return response.json() as Promise<T>;
  } finally {
    clearTimeout(timer);
  }
}

// Status
export async function getStatus(): Promise<AppStatus> {
  return request<AppStatus>("GET", "/api/v1/status");
}

export async function getConnectionStatus(): Promise<ConnectionStatus> {
  return request<ConnectionStatus>("GET", "/api/v1/status/connection");
}

// Core management
export async function startCore(): Promise<void> {
  return request<void>("POST", "/api/v1/core/start");
}

export async function stopCore(): Promise<void> {
  return request<void>("POST", "/api/v1/core/stop");
}

export async function restartCore(): Promise<void> {
  return request<void>("POST", "/api/v1/core/restart");
}

export async function recoverCore(): Promise<{ recovered: boolean }> {
  return request<{ recovered: boolean }>("POST", "/api/v1/core/recover");
}

// System proxy
export async function getSystemProxy(): Promise<{ enabled: boolean }> {
  return request<{ enabled: boolean }>("GET", "/api/v1/system-proxy");
}

export async function setSystemProxy(enabled: boolean): Promise<void> {
  return request<void>("POST", "/api/v1/system-proxy", { enabled });
}

// Subscription
export async function loadSubscription(url: string): Promise<void> {
  return request<void>("POST", "/api/v1/subscription", { url });
}

// Proxy groups
export async function getProxyGroups(): Promise<ProxyGroupView[]> {
  return request<ProxyGroupView[]>("GET", "/api/v1/proxy-groups");
}

export async function selectProxy(groupName: string, proxyName: string): Promise<void> {
  return request<void>("PUT", `/api/v1/proxy-groups/${encodeURIComponent(groupName)}/select`, { proxyName });
}

// Auto-stable
export async function getAutoStableStatus(): Promise<AutoStableStatus> {
  return request<AutoStableStatus>("GET", "/api/v1/autostable/status");
}

export async function setAutoStableEnabled(enabled: boolean): Promise<void> {
  return request<void>("PUT", "/api/v1/autostable/enabled", { enabled });
}

export async function runAutoStableTick(): Promise<AutoStableActionResult> {
  return request<AutoStableActionResult>("POST", "/api/v1/autostable/tick");
}

export async function selectAutoStableProxy(groupName: string): Promise<AutoStableActionResult> {
  return request<AutoStableActionResult>("POST", "/api/v1/autostable/select", { groupName });
}

// Logs
export async function getLogs(limit = 100, level?: string, query?: string): Promise<LogLine[]> {
  const params = new URLSearchParams({ limit: String(limit) });
  if (level) params.set("level", level);
  if (query) params.set("query", query);
  return request<LogLine[]>("GET", `/api/v1/logs?${params.toString()}`);
}

// Rule management
export async function getRules(): Promise<Rule[]> {
  return request<Rule[]>("GET", "/api/v1/rules");
}

export async function createRule(rule: Partial<Rule>): Promise<Rule> {
  return request<Rule>("POST", "/api/v1/rules", rule);
}

export async function updateRule(id: string, rule: Partial<Rule>): Promise<Rule> {
  return request<Rule>("PUT", `/api/v1/rules/${encodeURIComponent(id)}`, rule);
}

export async function deleteRule(id: string): Promise<void> {
  return request<void>("DELETE", `/api/v1/rules/${encodeURIComponent(id)}`);
}

export async function reorderRules(ruleIds: string[]): Promise<Rule[]> {
  return request<Rule[]>("POST", "/api/v1/rules/reorder", { ruleIds });
}

export async function validateRules(): Promise<RuleValidationError[]> {
  return request<RuleValidationError[]>("GET", "/api/v1/rules/validate");
}

// Rule templates
export async function getRuleTemplates(): Promise<RuleTemplate[]> {
  return request<RuleTemplate[]>("GET", "/api/v1/rule-templates");
}

export async function applyRuleTemplate(templateId: string, targetGroup?: string): Promise<Rule[]> {
  return request<Rule[]>("POST", `/api/v1/rule-templates/${encodeURIComponent(templateId)}/apply`, { targetGroup });
}

// Profile persistence
export async function listProfiles(): Promise<ProfileMeta[]> {
  return request<ProfileMeta[]>("GET", "/api/v1/profiles");
}

export async function saveProfile(): Promise<void> {
  return request<void>("POST", "/api/v1/profiles/save");
}

export async function loadProfile(id: string): Promise<void> {
  return request<void>("POST", `/api/v1/profiles/${encodeURIComponent(id)}/load`);
}

export async function deleteProfile(id: string): Promise<void> {
  return request<void>("DELETE", `/api/v1/profiles/${encodeURIComponent(id)}`);
}
