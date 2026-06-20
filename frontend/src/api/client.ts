export type AppStatus = {
  coreRunning: boolean;
  systemProxyEnabled: boolean;
  autoStableEnabled: boolean;
  activeProfileName: string;
  controllerAddress: string;
  lastError?: string;
};

export type ProxyView = {
  name: string;
  type?: string;
  latencyMs?: number;
  alive: boolean;
};

export type ProxyGroupView = {
  name: string;
  type: string;
  selected: string;
  proxies: ProxyView[];
};

export type LogLine = {
  time: string;
  level: string;
  message: string;
};

export type AutoStableNodeHealth = {
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
};

export type AutoStableGroupHealth = {
  name: string;
  type: string;
  selected?: string;
  proxies: AutoStableNodeHealth[];
};

export type AutoStableStatus = {
  enabled: boolean;
  available: boolean;
  running: boolean;
  lastTickAt?: string;
  lastAction?: string;
  lastSelected?: string;
  lastError?: string;
  health: AutoStableGroupHealth[];
};

export type AutoStableActionResult = {
  action: string;
  groupName?: string;
  selected?: string;
  changed: boolean;
  message?: string;
  completedAt: string;
  health?: AutoStableGroupHealth[];
};

type WailsApp = {
  GetAppStatus(): Promise<AppStatus>;
  StartCore(): Promise<void>;
  StopCore(): Promise<void>;
  RestartCore(): Promise<void>;
  SetSystemProxy(enabled: boolean): Promise<void>;
  LoadSubscription(url: string): Promise<void>;
  GetProxyGroups(): Promise<ProxyGroupView[]>;
  SelectProxy(groupName: string, proxyName: string): Promise<void>;
  GetLogs(limit: number): Promise<LogLine[]>;
  GetAutoStableStatus(): Promise<AutoStableStatus>;
  SetAutoStableEnabled(enabled: boolean): Promise<void>;
  RunAutoStableTick(): Promise<AutoStableActionResult>;
  SelectAutoStableProxy(groupName: string): Promise<AutoStableActionResult>;
};

declare global {
  interface Window {
    go?: {
      main?: {
        App?: WailsApp;
      };
    };
  }
}

function app(): WailsApp {
  const binding = window.go?.main?.App;
  if (!binding) {
    throw new Error("Wails backend binding is not available");
  }
  return binding;
}

export async function getStatus(): Promise<AppStatus> {
  return app().GetAppStatus();
}

export async function startCore(): Promise<void> {
  return app().StartCore();
}

export async function stopCore(): Promise<void> {
  return app().StopCore();
}

export async function restartCore(): Promise<void> {
  return app().RestartCore();
}

export async function setSystemProxy(enabled: boolean): Promise<void> {
  return app().SetSystemProxy(enabled);
}

export async function loadSubscription(url: string): Promise<void> {
  return app().LoadSubscription(url);
}

export async function getProxyGroups(): Promise<ProxyGroupView[]> {
  return app().GetProxyGroups();
}

export async function selectProxy(groupName: string, proxyName: string): Promise<void> {
  return app().SelectProxy(groupName, proxyName);
}

export async function getLogs(limit = 100): Promise<LogLine[]> {
  return app().GetLogs(limit);
}

export async function getAutoStableStatus(): Promise<AutoStableStatus> {
  return app().GetAutoStableStatus();
}

export async function setAutoStableEnabled(enabled: boolean): Promise<void> {
  return app().SetAutoStableEnabled(enabled);
}

export async function runAutoStableTick(): Promise<AutoStableActionResult> {
  return app().RunAutoStableTick();
}

export async function selectAutoStableProxy(groupName: string): Promise<AutoStableActionResult> {
  return app().SelectAutoStableProxy(groupName);
}
