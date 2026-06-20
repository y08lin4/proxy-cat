import { FormEvent, useEffect, useMemo, useState } from "react";
import {
  AppStatus,
  LogLine,
  ProxyGroupView,
  getLogs,
  getProxyGroups,
  getStatus,
  loadSubscription,
  restartCore,
  selectProxy,
  setSystemProxy,
  startCore,
  stopCore
} from "./api/client";

const emptyStatus: AppStatus = {
  coreRunning: false,
  systemProxyEnabled: false,
  activeProfileName: "",
  controllerAddress: "127.0.0.1:9090"
};

export default function App() {
  const [status, setStatus] = useState<AppStatus>(emptyStatus);
  const [groups, setGroups] = useState<ProxyGroupView[]>([]);
  const [logs, setLogs] = useState<LogLine[]>([]);
  const [subscriptionUrl, setSubscriptionUrl] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function refresh() {
    try {
      const [nextStatus, nextGroups, nextLogs] = await Promise.all([
        getStatus(),
        getProxyGroups(),
        getLogs(80)
      ]);
      setStatus(nextStatus);
      setGroups(nextGroups);
      setLogs(nextLogs);
      setError(nextStatus.lastError || null);
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
          <a href="#logs">Logs</a>
          <a href="#settings">Settings</a>
        </nav>
      </aside>

      <section className="workspace">
        <header className="topbar" id="dashboard">
          <div>
            <p className="eyebrow">Phase 1 MVP</p>
            <h2>Proxy control panel</h2>
          </div>
          <div className="toolbar">
            <button disabled={busy || status.coreRunning} onClick={() => run(startCore)} title="Start core">
              Start
            </button>
            <button disabled={busy || !status.coreRunning} onClick={() => run(stopCore)} title="Stop core">
              Stop
            </button>
            <button disabled={busy} onClick={() => run(restartCore)} title="Restart core">
              Restart
            </button>
          </div>
        </header>

        {error ? <div className="error-banner">{error}</div> : null}

        <section className="status-grid">
          <StatusCard label="Core" value={status.coreRunning ? "Running" : "Stopped"} tone={status.coreRunning ? "good" : "muted"} />
          <StatusCard label="System proxy" value={status.systemProxyEnabled ? "On" : "Off"} tone={status.systemProxyEnabled ? "good" : "muted"} />
          <StatusCard label="Profile" value={status.activeProfileName || "None"} tone={status.activeProfileName ? "good" : "muted"} />
          <StatusCard label="Controller" value={status.controllerAddress || "127.0.0.1:9090"} tone="muted" />
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
          </section>

          <section className="panel">
            <div className="panel-header">
              <h3>Connection</h3>
              <span>{selectedCount} selected</span>
            </div>
            <div className="connection-copy">
              <p>Traffic is routed by Mihomo groups generated from the active profile.</p>
              <p>Phase 1 keeps selection manual or Mihomo-native url-test.</p>
            </div>
          </section>
        </section>

        <section className="panel" id="proxies">
          <div className="panel-header">
            <h3>Proxy groups</h3>
            <button disabled={busy} onClick={refresh} title="Refresh status">Refresh</button>
          </div>
          {groups.length === 0 ? (
            <div className="empty-state">Load a subscription to generate PROXY and AUTO groups.</div>
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

function StatusCard(props: { label: string; value: string; tone: "good" | "muted" }) {
  return (
    <article className={`status-card ${props.tone}`}>
      <span>{props.label}</span>
      <strong>{props.value}</strong>
    </article>
  );
}

