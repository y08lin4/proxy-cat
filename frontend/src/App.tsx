import { FormEvent, useEffect, useMemo, useState } from "react";
import {
  AppStatus,
  AutoStableGroupHealth,
  AutoStableNodeHealth,
  AutoStableStatus,
  ConnectionStatus,
  LogLine,
  ProxyGroupView,
  getAutoStableStatus,
  getConnectionStatus,
  getLogs,
  getProxyGroups,
  getStatus,
  loadSubscription,
  restartCore,
  runAutoStableTick,
  selectProxy,
  setAutoStableEnabled,
  setSystemProxy,
  startCore,
  stopCore
} from "./api/client";

const emptyStatus: AppStatus = {
  coreRunning: false,
  systemProxyEnabled: false,
  autoStableEnabled: false,
  activeProfileName: "",
  controllerAddress: "127.0.0.1:9090"
};

const emptyAutoStable: AutoStableStatus = {
  enabled: false,
  available: false,
  running: false,
  health: []
};

const emptyConnection: ConnectionStatus = {
  coreRunning: false,
  uploadTotal: 0,
  downloadTotal: 0,
  connectionCount: 0
};

export default function App() {
  const [status, setStatus] = useState<AppStatus>(emptyStatus);
  const [autoStable, setAutoStable] = useState<AutoStableStatus>(emptyAutoStable);
  const [connection, setConnection] = useState<ConnectionStatus>(emptyConnection);
  const [groups, setGroups] = useState<ProxyGroupView[]>([]);
  const [logs, setLogs] = useState<LogLine[]>([]);
  const [subscriptionUrl, setSubscriptionUrl] = useState("");
  const [autoStableGroup, setAutoStableGroup] = useState("AUTO-STABLE");
  const [proxyQuery, setProxyQuery] = useState("");
  const [proxyView, setProxyView] = useState<"all" | "selected" | "auto">("all");
  const [logLevel, setLogLevel] = useState("all");
  const [logQuery, setLogQuery] = useState("");
  const [busy, setBusy] = useState(false);
  const [healthBusy, setHealthBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function refresh() {
    try {
      const [nextStatus, nextGroups, nextLogs, nextAutoStable, nextConnection] = await Promise.all([
        getStatus(),
        getProxyGroups(),
        getLogs(80),
        getAutoStableStatus(),
        getConnectionStatus()
      ]);
      setStatus(nextStatus);
      setGroups(nextGroups);
      setLogs(nextLogs);
      setAutoStable(nextAutoStable);
      setConnection(nextConnection);
      setError(nextStatus.lastError || nextAutoStable.lastError || null);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  }

  async function run(action: () => Promise<void>) {
    setBusy(true);
    setError(null);
    try {
      await action();
      await refresh();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
      await refresh();
    } finally {
      setBusy(false);
    }
  }

  async function refreshHealth() {
    setHealthBusy(true);
    setError(null);
    try {
      setAutoStable(await getAutoStableStatus());
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setHealthBusy(false);
    }
  }

  async function submitSubscription(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const url = subscriptionUrl.trim();
    if (!url) {
      setError("Subscription URL is required");
      return;
    }
    await run(() => loadSubscription(url));
  }

  useEffect(() => {
    refresh();
    const id = window.setInterval(refresh, 5000);
    return () => window.clearInterval(id);
  }, []);

  const selectedCount = useMemo(
    () => groups.filter((group) => group.selected).length,
    [groups]
  );
  const selectedNodes = useMemo(
    () => groups.filter((group) => group.selected).map((group) => `${group.name}: ${group.selected}`),
    [groups]
  );
  const autoStableGroups = useMemo(
    () => groups.filter((group) => group.type === "auto-stable" || group.name.toLowerCase().includes("stable")),
    [groups]
  );
  const healthRows = useMemo(
    () => flattenHealth(autoStable.health).sort((left, right) => healthScore(left.node) - healthScore(right.node)),
    [autoStable.health]
  );
  const healthSummary = useMemo(() => summarizeHealth(healthRows), [healthRows]);
  const filteredGroups = useMemo(
    () => filterGroups(groups, proxyQuery, proxyView),
    [groups, proxyQuery, proxyView]
  );
  const logLevels = useMemo(() => uniqueLogLevels(logs), [logs]);
  const filteredLogs = useMemo(
    () => filterLogs(logs, logLevel, logQuery),
    [logs, logLevel, logQuery]
  );
  const connectionTone = connection.coreRunning ? (connection.connectionCount > 0 ? "good" : "warn") : "muted";
  const connectionLabel = connection.coreRunning ? (connection.connectionCount > 0 ? "Connected" : "Core ready") : "Offline";

  useEffect(() => {
    if (autoStableGroups.length > 0 && !autoStableGroups.some((group) => group.name === autoStableGroup)) {
      setAutoStableGroup(autoStableGroups[0].name);
    }
  }, [autoStableGroup, autoStableGroups]);

  return (
    <main className="app-shell">
      <aside className="sidebar">
        <div className="brand">
          <div className="brand-mark">PC</div>
          <div>
            <h1>Proxy-Cat</h1>
            <p>Mihomo client</p>
          </div>
        </div>
        <nav className="nav-list">
          <a className="active" href="#dashboard">Dashboard</a>
          <a href="#proxies">Proxies</a>
          <a href="#auto-stable">Auto-stable</a>
          <a href="#logs">Logs</a>
          <a href="#settings">Settings</a>
        </nav>
      </aside>

      <section className="workspace">
        <header className="topbar" id="dashboard">
          <div>
            <p className="eyebrow">Phase 3</p>
            <h2>Proxy control panel</h2>
          </div>
          <div className="toolbar">
            <button disabled={busy || status.coreRunning} onClick={() => run(startCore)} title="Start core">Start</button>
            <button disabled={busy || !status.coreRunning} onClick={() => run(stopCore)} title="Stop core">Stop</button>
            <button disabled={busy} onClick={() => run(restartCore)} title="Restart core">Restart</button>
          </div>
        </header>

        {error ? <div className="error-banner">{error}</div> : null}

        <section className="status-grid">
          <StatusCard label="Core" value={status.coreRunning ? "Running" : "Stopped"} tone={status.coreRunning ? "good" : "muted"} />
          <StatusCard label="System proxy" value={status.systemProxyEnabled ? "On" : "Off"} tone={status.systemProxyEnabled ? "good" : "muted"} />
          <StatusCard label="Auto-stable" value={autoStable.enabled ? "On" : "Off"} tone={autoStable.enabled ? "good" : "muted"} />
          <StatusCard label="Profile" value={status.activeProfileName || "None"} tone={status.activeProfileName ? "good" : "muted"} />
        </section>

        <section className="connection-strip" aria-label="Connection status">
          <div className={`connection-status ${connectionTone}`}>
            <span className="status-dot" />
            <div>
              <strong>{connectionLabel}</strong>
              <span>{status.controllerAddress}</span>
            </div>
          </div>
          <Metric label="Active profile" value={status.activeProfileName || "No profile"} />
          <Metric label="Connections" value={`${connection.connectionCount}`} />
          <Metric label="Traffic" value={`${formatBytes(connection.downloadTotal)} down / ${formatBytes(connection.uploadTotal)} up`} />
          <Metric label="Auto-stable scan" value={autoStable.running || healthBusy ? "Running" : autoStable.lastTickAt ? formatRelativeTime(autoStable.lastTickAt) : "Idle"} />
          <Metric label="Last action" value={autoStable.lastAction || autoStable.lastSelected || "None"} />
        </section>

        <section className="panel-row">
          <section className="panel" id="settings">
            <div className="panel-header">
              <h3>Subscription</h3>
              <span>{groups.length} groups</span>
            </div>
            <form className="subscription-form" onSubmit={submitSubscription}>
              <input
                value={subscriptionUrl}
                onChange={(event) => setSubscriptionUrl(event.target.value)}
                placeholder="https://example.com/subscription.yaml"
                spellCheck={false}
              />
              <button disabled={busy} type="submit">Load</button>
            </form>
            <label className="switch-row">
              <span>Windows system proxy</span>
              <input
                type="checkbox"
                checked={status.systemProxyEnabled}
                onChange={(event) => run(() => setSystemProxy(event.target.checked))}
                disabled={busy}
              />
            </label>
            <label className="switch-row">
              <span>Auto-stable node selection</span>
              <input
                type="checkbox"
                checked={autoStable.enabled}
                onChange={(event) => run(() => setAutoStableEnabled(event.target.checked))}
                disabled={busy || !autoStable.available}
              />
            </label>
          </section>

          <section className="panel">
            <div className="panel-header">
              <h3>Connection</h3>
              <span>{selectedCount} selected</span>
            </div>
            <div className="connection-list">
              {selectedNodes.length === 0 ? (
                <div className="empty-state compact">No selected proxy groups yet.</div>
              ) : (
                selectedNodes.slice(0, 4).map((item) => <span key={item}>{item}</span>)
              )}
              {selectedNodes.length > 4 ? <small>+{selectedNodes.length - 4} more groups</small> : null}
            </div>
          </section>
        </section>

        <section className="panel" id="auto-stable">
          <div className="panel-header">
            <div>
              <h3>Auto-stable health</h3>
              <span>{healthRows.length} nodes</span>
            </div>
            <div className="health-actions">
              <select
                value={autoStableGroup}
                onChange={(event) => setAutoStableGroup(event.target.value)}
                disabled={busy || autoStableGroups.length === 0}
              >
                {autoStableGroups.length === 0 ? (
                  <option value="AUTO-STABLE">AUTO-STABLE</option>
                ) : (
                  autoStableGroups.map((group) => (
                    <option key={group.name} value={group.name}>{group.name}</option>
                  ))
                )}
              </select>
              <button disabled={busy || healthBusy || !autoStable.available} onClick={refreshHealth} title="Refresh health">Refresh</button>
              <button disabled={busy || healthBusy || !autoStable.available || !autoStable.enabled} onClick={() => run(async () => { await runAutoStableTick(); })} title="Run auto-stable">Tick</button>
            </div>
          </div>
          <div className="scan-summary">
            <Metric label="Best node" value={healthSummary.bestNode || "n/a"} />
            <Metric label="Healthy" value={`${healthSummary.healthy}/${healthSummary.total}`} />
            <Metric label="Avg latency" value={healthSummary.averageLatency ? `${healthSummary.averageLatency} ms` : "n/a"} />
            <Metric label="Failure checks" value={`${healthSummary.failures}`} />
          </div>
          {healthRows.length === 0 ? (
            <div className="empty-state">
              {autoStable.available ? "No health samples yet." : "Load a subscription to create AUTO-STABLE."}
            </div>
          ) : (
            <div className="health-table">
              <div className="health-row health-head">
                <span>Node</span>
                <span>Score</span>
                <span>Latency</span>
                <span>Failure</span>
                <span>Checked</span>
              </div>
              {healthRows.map((row) => (
                <div className="health-row" key={`${row.groupName}-${row.node.name}`}>
                  <strong>{row.node.name}</strong>
                  <span>{formatScore(row.node)}</span>
                  <span className={row.node.alive ? "cell-good" : "cell-bad"}>{formatLatency(row.node)}</span>
                  <span>{formatFailureRate(row.node)}</span>
                  <span>{formatRelativeTime(row.node.lastCheckedAt)}</span>
                </div>
              ))}
            </div>
          )}
        </section>

        <section className="panel" id="proxies">
          <div className="panel-header">
            <div>
              <h3>Proxy groups</h3>
              <span>{filteredGroups.length}/{groups.length} visible</span>
            </div>
            <div className="proxy-actions">
              <input
                value={proxyQuery}
                onChange={(event) => setProxyQuery(event.target.value)}
                placeholder="Search group or node"
                spellCheck={false}
              />
              <select value={proxyView} onChange={(event) => setProxyView(event.target.value as "all" | "selected" | "auto")}>
                <option value="all">All groups</option>
                <option value="selected">Selected</option>
                <option value="auto">Auto groups</option>
              </select>
              <button disabled={busy} onClick={refresh} title="Refresh status">Refresh</button>
            </div>
          </div>
          {groups.length === 0 ? (
            <div className="empty-state">Load a subscription to generate PROXY, AUTO-STABLE, and AUTO groups.</div>
          ) : filteredGroups.length === 0 ? (
            <div className="empty-state">No proxy groups match the current filter.</div>
          ) : (
            <div className="group-list">
              {filteredGroups.map((group) => (
                <article className="group-card" key={group.name}>
                  <div className="group-title">
                    <div>
                      <h4>{group.name}</h4>
                      <span>{group.type}</span>
                    </div>
                    <strong>{group.selected || "Not selected"}</strong>
                  </div>
                  <div className="node-list">
                    {group.proxies.map((proxy) => (
                      <button
                        className={proxy.name === group.selected ? "node active" : "node"}
                        key={proxy.name}
                        disabled={busy}
                        onClick={() => run(() => selectProxy(group.name, proxy.name))}
                        title={`Select ${proxy.name}`}
                      >
                        <span>{proxy.name}</span>
                        <small>
                          {proxy.type || "group"}
                          {typeof proxy.latencyMs === "number" && proxy.latencyMs > 0 ? ` / ${proxy.latencyMs} ms` : ""}
                          {!proxy.alive ? " / down" : ""}
                        </small>
                      </button>
                    ))}
                  </div>
                </article>
              ))}
            </div>
          )}
        </section>

        <section className="panel" id="logs">
          <div className="panel-header">
            <div>
              <h3>Logs</h3>
              <span>{filteredLogs.length}/{logs.length} lines</span>
            </div>
            <div className="log-controls">
              <select value={logLevel} onChange={(event) => setLogLevel(event.target.value)}>
                <option value="all">All levels</option>
                {logLevels.map((level) => (
                  <option key={level} value={level}>{level}</option>
                ))}
              </select>
              <input
                value={logQuery}
                onChange={(event) => setLogQuery(event.target.value)}
                placeholder="Filter logs"
                spellCheck={false}
              />
              <button
                disabled={logLevel === "all" && logQuery.length === 0}
                onClick={() => {
                  setLogLevel("all");
                  setLogQuery("");
                }}
                title="Clear log filters"
              >
                Clear
              </button>
            </div>
          </div>
          <div className="log-list">
            {logs.length === 0 ? (
              <div className="empty-state">No logs yet.</div>
            ) : filteredLogs.length === 0 ? (
              <div className="empty-state">No log lines match the current filter.</div>
            ) : (
              filteredLogs.map((line, index) => (
                <div className={`log-line ${logTone(line.level)}`} key={`${line.time}-${index}`}>
                  <time>{new Date(line.time).toLocaleTimeString()}</time>
                  <span>{line.level}</span>
                  <p>{line.message}</p>
                </div>
              ))
            )}
          </div>
        </section>
      </section>
    </main>
  );
}

type HealthRow = {
  groupName: string;
  node: AutoStableNodeHealth;
};

function StatusCard(props: { label: string; value: string; tone: "good" | "muted" }) {
  return (
    <article className={`status-card ${props.tone}`}>
      <span>{props.label}</span>
      <strong>{props.value}</strong>
    </article>
  );
}

function Metric(props: { label: string; value: string }) {
  return (
    <div className="metric">
      <span>{props.label}</span>
      <strong title={props.value}>{props.value}</strong>
    </div>
  );
}

function flattenHealth(groups: AutoStableGroupHealth[]): HealthRow[] {
  return groups.flatMap((group) =>
    group.proxies.map((node) => ({
      groupName: group.name,
      node
    }))
  );
}

function filterGroups(groups: ProxyGroupView[], query: string, view: "all" | "selected" | "auto"): ProxyGroupView[] {
  const normalized = query.trim().toLowerCase();
  return groups.filter((group) => {
    if (view === "selected" && !group.selected) {
      return false;
    }
    if (view === "auto" && !isAutoGroup(group)) {
      return false;
    }
    if (!normalized) {
      return true;
    }
    return (
      group.name.toLowerCase().includes(normalized) ||
      group.type.toLowerCase().includes(normalized) ||
      group.selected.toLowerCase().includes(normalized) ||
      group.proxies.some((proxy) => proxy.name.toLowerCase().includes(normalized))
    );
  });
}

function isAutoGroup(group: ProxyGroupView): boolean {
  const name = group.name.toLowerCase();
  const type = group.type.toLowerCase();
  return type.includes("auto") || name.includes("auto") || name.includes("stable");
}

function uniqueLogLevels(logs: LogLine[]): string[] {
  return Array.from(new Set(logs.map((line) => line.level).filter(Boolean))).sort((left, right) => left.localeCompare(right));
}

function filterLogs(logs: LogLine[], level: string, query: string): LogLine[] {
  const normalized = query.trim().toLowerCase();
  return logs.filter((line) => {
    if (level !== "all" && line.level !== level) {
      return false;
    }
    if (!normalized) {
      return true;
    }
    return `${line.level} ${line.message}`.toLowerCase().includes(normalized);
  });
}

function summarizeHealth(rows: HealthRow[]) {
  const latencies = rows
    .map((row) => row.node.latencyMs)
    .filter((latency): latency is number => typeof latency === "number" && latency > 0);
  const best = rows.find((row) => Number.isFinite(healthScore(row.node)));
  const failures = rows.reduce((total, row) => total + (row.node.failureCount || 0), 0);

  return {
    total: rows.length,
    healthy: rows.filter((row) => row.node.alive).length,
    failures,
    bestNode: best?.node.name,
    averageLatency: latencies.length > 0 ? Math.round(latencies.reduce((total, latency) => total + latency, 0) / latencies.length) : 0
  };
}

function healthScore(row: AutoStableNodeHealth): number {
  return typeof row.score === "number" && Number.isFinite(row.score) ? row.score : Number.POSITIVE_INFINITY;
}

function formatScore(row: AutoStableNodeHealth): string {
  const score = healthScore(row);
  return Number.isFinite(score) ? score.toFixed(0) : "n/a";
}

function formatLatency(row: AutoStableNodeHealth): string {
  return typeof row.latencyMs === "number" && row.latencyMs > 0 ? `${row.latencyMs} ms` : "n/a";
}

function formatFailureRate(row: AutoStableNodeHealth): string {
  const total = row.totalChecks ?? ((row.successCount || 0) + (row.failureCount || 0));
  if (!total) {
    return "n/a";
  }
  const rate = row.failureRate ?? ((row.failureCount || 0) / total);
  return `${(rate * 100).toFixed(0)}%`;
}

function formatRelativeTime(value?: string): string {
  if (!value) {
    return "n/a";
  }
  const time = new Date(value).getTime();
  if (!Number.isFinite(time)) {
    return "n/a";
  }
  const seconds = Math.max(0, Math.round((Date.now() - time) / 1000));
  if (seconds < 60) {
    return `${seconds}s ago`;
  }
  const minutes = Math.round(seconds / 60);
  if (minutes < 60) {
    return `${minutes}m ago`;
  }
  const hours = Math.round(minutes / 60);
  return `${hours}h ago`;
}

function formatBytes(value: number): string {
  if (!Number.isFinite(value) || value <= 0) {
    return "0 B";
  }
  const units = ["B", "KB", "MB", "GB"];
  let size = value;
  let unitIndex = 0;
  while (size >= 1024 && unitIndex < units.length - 1) {
    size = size / 1024;
    unitIndex++;
  }
  const digits = unitIndex === 0 ? 0 : 1;
  return `${size.toFixed(digits)} ${units[unitIndex]}`;
}

function logTone(level: string): string {
  const normalized = level.toLowerCase();
  if (normalized.includes("error") || normalized.includes("fatal")) {
    return "bad";
  }
  if (normalized.includes("warn")) {
    return "warn";
  }
  if (normalized.includes("debug") || normalized.includes("trace")) {
    return "muted";
  }
  return "info";
}
