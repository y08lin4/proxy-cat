import { FormEvent, useEffect, useMemo, useState } from "react";
import {
  AppStatus,
  AutoStableGroupHealth,
  AutoStableNodeHealth,
  AutoStableStatus,
  LogLine,
  ProxyGroupView,
  getAutoStableStatus,
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

export default function App() {
  const [status, setStatus] = useState<AppStatus>(emptyStatus);
  const [autoStable, setAutoStable] = useState<AutoStableStatus>(emptyAutoStable);
  const [groups, setGroups] = useState<ProxyGroupView[]>([]);
  const [logs, setLogs] = useState<LogLine[]>([]);
  const [subscriptionUrl, setSubscriptionUrl] = useState("");
  const [autoStableGroup, setAutoStableGroup] = useState("AUTO-STABLE");
  const [busy, setBusy] = useState(false);
  const [healthBusy, setHealthBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function refresh() {
    try {
      const [nextStatus, nextGroups, nextLogs, nextAutoStable] = await Promise.all([
        getStatus(),
        getProxyGroups(),
        getLogs(80),
        getAutoStableStatus()
      ]);
      setStatus(nextStatus);
      setGroups(nextGroups);
      setLogs(nextLogs);
      setAutoStable(nextAutoStable);
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
  const autoStableGroups = useMemo(
    () => groups.filter((group) => group.type === "auto-stable" || group.name.toLowerCase().includes("stable")),
    [groups]
  );
  const healthRows = useMemo(
    () => flattenHealth(autoStable.health).sort((left, right) => healthScore(left.node) - healthScore(right.node)),
    [autoStable.health]
  );

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
            <p className="eyebrow">Phase 2</p>
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
            <div className="connection-copy">
              <p>Traffic is routed by Mihomo groups generated from the active profile.</p>
              <p>{autoStable.available ? "Auto-stable selects the managed group from cached health scores." : "Load a subscription to enable auto-stable."}</p>
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
              </div>
              {healthRows.map((row) => (
                <div className="health-row" key={`${row.groupName}-${row.node.name}`}>
                  <strong>{row.node.name}</strong>
                  <span>{formatScore(row.node)}</span>
                  <span>{formatLatency(row.node)}</span>
                  <span>{formatFailureRate(row.node)}</span>
                </div>
              ))}
            </div>
          )}
        </section>

        <section className="panel" id="proxies">
          <div className="panel-header">
            <h3>Proxy groups</h3>
            <button disabled={busy} onClick={refresh} title="Refresh status">Refresh</button>
          </div>
          {groups.length === 0 ? (
            <div className="empty-state">Load a subscription to generate PROXY, AUTO-STABLE, and AUTO groups.</div>
          ) : (
            <div className="group-list">
              {groups.map((group) => (
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
                        <small>{proxy.type || "group"}</small>
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
            <h3>Logs</h3>
            <span>{logs.length} lines</span>
          </div>
          <div className="log-list">
            {logs.length === 0 ? (
              <div className="empty-state">No logs yet.</div>
            ) : (
              logs.map((line, index) => (
                <div className="log-line" key={`${line.time}-${index}`}>
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

function flattenHealth(groups: AutoStableGroupHealth[]): HealthRow[] {
  return groups.flatMap((group) =>
    group.proxies.map((node) => ({
      groupName: group.name,
      node
    }))
  );
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
