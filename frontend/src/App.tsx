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

type ViewId = "overview" | "proxies" | "auto" | "logs" | "settings";

const navItems: Array<{ id: ViewId; label: string; mark: string }> = [
  { id: "overview", label: "概览", mark: "览" },
  { id: "proxies", label: "代理", mark: "代" },
  { id: "auto", label: "自动选择", mark: "稳" },
  { id: "logs", label: "日志", mark: "志" },
  { id: "settings", label: "设置", mark: "设" }
];

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
  const [activeView, setActiveView] = useState<ViewId>("overview");
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
      const normalizedStatus = normalizeAppStatus(nextStatus);
      const normalizedAutoStable = normalizeAutoStable(nextAutoStable);
      setStatus(normalizedStatus);
      setGroups(normalizeGroups(nextGroups));
      setLogs(normalizeLogs(nextLogs));
      setAutoStable(normalizedAutoStable);
      setConnection(normalizeConnection(nextConnection));
      setError(normalizedStatus.lastError || normalizedAutoStable.lastError || null);
    } catch (err) {
      setError(toMessage(err));
    }
  }

  async function run(action: () => Promise<void>) {
    setBusy(true);
    setError(null);
    try {
      await action();
      await refresh();
    } catch (err) {
      setError(toMessage(err));
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
      setError(toMessage(err));
    } finally {
      setHealthBusy(false);
    }
  }

  async function submitSubscription(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const url = subscriptionUrl.trim();
    if (!url) {
      setError("请输入订阅地址");
      return;
    }
    await run(() => loadSubscription(url));
  }

  useEffect(() => {
    refresh();
    const id = window.setInterval(refresh, 5000);
    return () => window.clearInterval(id);
  }, []);

  const selectedNodes = useMemo(
    () => groups.filter((group) => group.selected).map((group) => `${group.name}：${group.selected}`),
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
  const activeGroup = useMemo(
    () => groups.find((group) => group.name === autoStableGroup) || autoStableGroups[0] || groups[0],
    [autoStableGroup, autoStableGroups, groups]
  );

  useEffect(() => {
    if (autoStableGroups.length > 0 && !autoStableGroups.some((group) => group.name === autoStableGroup)) {
      setAutoStableGroup(autoStableGroups[0].name);
    }
  }, [autoStableGroup, autoStableGroups]);

  return (
    <main className="app-shell">
      <aside className="side-rail" aria-label="主导航">
        <div className="brand-block">
          <div className="cat-mark">PC</div>
          <div>
            <h1>Proxy-Cat</h1>
            <p>更稳定的 Mihomo 客户端</p>
          </div>
        </div>

        <nav className="nav-stack">
          {navItems.map((item) => (
            <button
              className={activeView === item.id ? "nav-item active" : "nav-item"}
              key={item.id}
              onClick={() => setActiveView(item.id)}
              type="button"
            >
              <span>{item.mark}</span>
              {item.label}
            </button>
          ))}
        </nav>

        <div className="rail-footer">
          <span className={status.coreRunning ? "soft-pill online" : "soft-pill"}>{status.coreRunning ? "内核运行中" : "内核未启动"}</span>
          <small>{status.controllerAddress}</small>
        </div>
      </aside>

      <section className="workspace">
        <header className="topbar">
          <div>
            <p>当前配置</p>
            <h2>{status.activeProfileName || "未加载订阅"}</h2>
          </div>
          <div className="top-actions">
            <button className="ghost-button" disabled={busy} onClick={refresh} type="button">刷新</button>
            <button className="ghost-button" disabled={busy || !status.coreRunning} onClick={() => run(stopCore)} type="button">停止</button>
            <button className="ghost-button" disabled={busy} onClick={() => run(restartCore)} type="button">重启</button>
            <button className="primary-button" disabled={busy || status.coreRunning} onClick={() => run(startCore)} type="button">启动内核</button>
          </div>
        </header>

        {error ? <div className="notice error">{error}</div> : null}

        <section className="content-grid">
          <div className="main-column">
            {activeView === "overview" ? (
              <OverviewView
                autoStable={autoStable}
                connection={connection}
                groups={groups}
                healthSummary={healthSummary}
                logs={logs}
                selectedNodes={selectedNodes}
                status={status}
              />
            ) : null}

            {activeView === "proxies" ? (
              <ProxyViewPanel
                busy={busy}
                filteredGroups={filteredGroups}
                groups={groups}
                proxyQuery={proxyQuery}
                proxyView={proxyView}
                setProxyQuery={setProxyQuery}
                setProxyView={setProxyView}
                select={(groupName, proxyName) => run(() => selectProxy(groupName, proxyName))}
              />
            ) : null}

            {activeView === "auto" ? (
              <AutoStableView
                activeGroup={activeGroup}
                autoStable={autoStable}
                autoStableGroup={autoStableGroup}
                autoStableGroups={autoStableGroups}
                busy={busy}
                healthBusy={healthBusy}
                healthRows={healthRows}
                healthSummary={healthSummary}
                refreshHealth={refreshHealth}
                runTick={() => run(async () => {
                  await runAutoStableTick();
                })}
                setAutoStableGroup={setAutoStableGroup}
                toggle={(enabled) => run(() => setAutoStableEnabled(enabled))}
              />
            ) : null}

            {activeView === "logs" ? (
              <LogsView
                filteredLogs={filteredLogs}
                logLevel={logLevel}
                logLevels={logLevels}
                logQuery={logQuery}
                logs={logs}
                setLogLevel={setLogLevel}
                setLogQuery={setLogQuery}
              />
            ) : null}

            {activeView === "settings" ? (
              <SettingsView
                autoStable={autoStable}
                busy={busy}
                status={status}
                subscriptionUrl={subscriptionUrl}
                setSubscriptionUrl={setSubscriptionUrl}
                submitSubscription={submitSubscription}
                toggleAuto={(enabled) => run(() => setAutoStableEnabled(enabled))}
                toggleSystemProxy={(enabled) => run(() => setSystemProxy(enabled))}
              />
            ) : null}
          </div>

          <ContextPanel
            autoStable={autoStable}
            busy={busy}
            connection={connection}
            groups={groups}
            healthSummary={healthSummary}
            logs={logs}
            selectedNodes={selectedNodes}
            status={status}
            start={() => run(startCore)}
            stop={() => run(stopCore)}
            restart={() => run(restartCore)}
            toggleAuto={(enabled) => run(() => setAutoStableEnabled(enabled))}
            toggleSystemProxy={(enabled) => run(() => setSystemProxy(enabled))}
          />
        </section>
      </section>
    </main>
  );
}

function ContextPanel(props: {
  autoStable: AutoStableStatus;
  busy: boolean;
  connection: ConnectionStatus;
  groups: ProxyGroupView[];
  healthSummary: ReturnType<typeof summarizeHealth>;
  logs: LogLine[];
  selectedNodes: string[];
  status: AppStatus;
  start(): void;
  stop(): void;
  restart(): void;
  toggleAuto(enabled: boolean): void;
  toggleSystemProxy(enabled: boolean): void;
}) {
  return (
    <aside className="context-column" aria-label="运行状态">
      <section className="context-card depth-card">
        <div className="context-head">
          <span className={props.status.coreRunning ? "state-chip good" : "state-chip"}>{props.status.coreRunning ? "运行中" : "未启动"}</span>
          <strong>运行控制</strong>
        </div>
        <div className="control-grid">
          <button className="primary-button" disabled={props.busy || props.status.coreRunning} onClick={props.start} type="button">启动</button>
          <button className="ghost-button" disabled={props.busy || !props.status.coreRunning} onClick={props.stop} type="button">停止</button>
          <button className="ghost-button" disabled={props.busy} onClick={props.restart} type="button">重启</button>
        </div>
      </section>

      <section className="context-card">
        <PanelTitle title="实时状态" meta={props.status.controllerAddress} />
        <div className="status-rows">
          <StatusLine label="系统代理" value={props.status.systemProxyEnabled ? "已开启" : "未开启"} good={props.status.systemProxyEnabled} />
          <StatusLine label="自动选择" value={props.autoStable.enabled ? "已开启" : "未开启"} good={props.autoStable.enabled} />
          <StatusLine label="连接数" value={`${props.connection.connectionCount}`} good={props.connection.connectionCount > 0} />
          <StatusLine label="代理组" value={`${props.groups.length} 个`} good={props.groups.length > 0} />
        </div>
      </section>

      <section className="context-card">
        <PanelTitle title="快捷开关" meta="本机" />
        <div className="toggle-list compact">
          <label className="switch-card">
            <span>系统代理</span>
            <input
              checked={props.status.systemProxyEnabled}
              disabled={props.busy}
              onChange={(event) => props.toggleSystemProxy(event.target.checked)}
              type="checkbox"
            />
          </label>
          <label className="switch-card">
            <span>自动选择</span>
            <input
              checked={props.autoStable.enabled}
              disabled={props.busy || !props.autoStable.available}
              onChange={(event) => props.toggleAuto(event.target.checked)}
              type="checkbox"
            />
          </label>
        </div>
      </section>

      <section className="context-card">
        <PanelTitle title="当前出口" meta={`${props.selectedNodes.length} 项`} />
        <div className="context-list">
          {props.selectedNodes.length === 0 ? (
            <span>暂无已选择分组</span>
          ) : (
            props.selectedNodes.slice(0, 4).map((item) => <span key={item}>{item}</span>)
          )}
        </div>
      </section>

      <section className="context-card">
        <PanelTitle title="稳定性" meta={props.healthSummary.bestNode || "暂无"} />
        <div className="status-rows">
          <StatusLine label="健康节点" value={`${props.healthSummary.healthy}/${props.healthSummary.total}`} good={props.healthSummary.healthy > 0} />
          <StatusLine label="平均延迟" value={props.healthSummary.averageLatency ? `${props.healthSummary.averageLatency} ms` : "暂无"} />
          <StatusLine label="最近日志" value={`${props.logs.length} 条`} />
        </div>
      </section>
    </aside>
  );
}

function StatusLine(props: { label: string; value: string; good?: boolean }) {
  return (
    <div className={props.good ? "status-line good" : "status-line"}>
      <span>{props.label}</span>
      <strong>{props.value}</strong>
    </div>
  );
}

function OverviewView(props: {
  autoStable: AutoStableStatus;
  connection: ConnectionStatus;
  groups: ProxyGroupView[];
  healthSummary: ReturnType<typeof summarizeHealth>;
  logs: LogLine[];
  selectedNodes: string[];
  status: AppStatus;
}) {
  const connectionLabel = props.connection.coreRunning
    ? props.connection.connectionCount > 0 ? "连接活跃" : "内核就绪"
    : "离线";

  return (
    <div className="view-stack">
      <section className="hero-panel">
        <div className="hero-copy">
          <span className={props.status.coreRunning ? "state-chip good" : "state-chip"}>{connectionLabel}</span>
          <h3>{props.status.coreRunning ? "代理服务已准备好" : "启动内核后开始代理"}</h3>
          <p>加载订阅、开启系统代理，再让自动稳定选择帮你减少手动切换。</p>
        </div>
        <div className="hero-metrics">
          <Metric label="连接数" value={`${props.connection.connectionCount}`} />
          <Metric label="下载" value={formatBytes(props.connection.downloadTotal)} />
          <Metric label="上传" value={formatBytes(props.connection.uploadTotal)} />
        </div>
      </section>

      <section className="quick-grid">
        <StatusTile label="内核" value={props.status.coreRunning ? "运行中" : "未启动"} tone={props.status.coreRunning ? "good" : "muted"} />
        <StatusTile label="系统代理" value={props.status.systemProxyEnabled ? "已开启" : "未开启"} tone={props.status.systemProxyEnabled ? "good" : "muted"} />
        <StatusTile label="自动选择" value={props.autoStable.enabled ? "已开启" : "未开启"} tone={props.autoStable.enabled ? "good" : "muted"} />
        <StatusTile label="代理组" value={`${props.groups.length} 个`} tone={props.groups.length > 0 ? "good" : "muted"} />
      </section>

      <section className="dashboard-grid">
        <article className="panel surface">
          <PanelTitle title="当前选择" meta={`${props.selectedNodes.length} 个分组`} />
          <div className="selected-list">
            {props.selectedNodes.length === 0 ? (
              <EmptyState text="还没有已选择的代理组" />
            ) : (
              props.selectedNodes.slice(0, 6).map((item) => <span key={item}>{item}</span>)
            )}
          </div>
        </article>

        <article className="panel surface">
          <PanelTitle title="稳定性摘要" meta={props.healthSummary.total > 0 ? "已有检测数据" : "等待检测"} />
          <div className="summary-list">
            <Metric label="最佳节点" value={props.healthSummary.bestNode || "暂无"} />
            <Metric label="健康节点" value={`${props.healthSummary.healthy}/${props.healthSummary.total}`} />
            <Metric label="平均延迟" value={props.healthSummary.averageLatency ? `${props.healthSummary.averageLatency} ms` : "暂无"} />
          </div>
        </article>

        <article className="panel surface">
          <PanelTitle title="最近日志" meta={`${props.logs.length} 条`} />
          <div className="mini-log-list">
            {props.logs.length === 0 ? (
              <EmptyState text="暂无日志" />
            ) : (
              props.logs.slice(0, 5).map((line, index) => (
                <div className="mini-log" key={`${line.time}-${index}`}>
                  <span>{translateLogLevel(line.level)}</span>
                  <p>{line.message}</p>
                </div>
              ))
            )}
          </div>
        </article>
      </section>
    </div>
  );
}

function ProxyViewPanel(props: {
  busy: boolean;
  filteredGroups: ProxyGroupView[];
  groups: ProxyGroupView[];
  proxyQuery: string;
  proxyView: "all" | "selected" | "auto";
  setProxyQuery(value: string): void;
  setProxyView(value: "all" | "selected" | "auto"): void;
  select(groupName: string, proxyName: string): void;
}) {
  return (
    <section className="panel full-panel">
      <PanelTitle title="代理" meta={`${props.filteredGroups.length}/${props.groups.length} 个分组`} />
      <div className="filter-bar">
        <input
          value={props.proxyQuery}
          onChange={(event) => props.setProxyQuery(event.target.value)}
          placeholder="搜索分组或节点"
          spellCheck={false}
        />
        <select value={props.proxyView} onChange={(event) => props.setProxyView(event.target.value as "all" | "selected" | "auto")}>
          <option value="all">全部分组</option>
          <option value="selected">已选择</option>
          <option value="auto">自动分组</option>
        </select>
      </div>

      {props.groups.length === 0 ? (
        <EmptyState text="先在设置中加载订阅，随后这里会出现代理组和节点" />
      ) : props.filteredGroups.length === 0 ? (
        <EmptyState text="没有匹配当前筛选条件的代理组" />
      ) : (
        <div className="proxy-grid">
          {props.filteredGroups.map((group) => (
            <article className="proxy-card" key={group.name}>
              <div className="proxy-card-head">
                <div>
                  <h3>{group.name}</h3>
                  <span>{translateGroupType(group.type)} · {group.proxies.length} 个节点</span>
                </div>
                <strong>{group.selected || "未选择"}</strong>
              </div>
              <div className="node-grid">
                {group.proxies.map((proxy) => (
                  <button
                    className={proxy.name === group.selected ? "node-card active" : "node-card"}
                    disabled={props.busy}
                    key={proxy.name}
                    onClick={() => props.select(group.name, proxy.name)}
                    type="button"
                  >
                    <span>{proxy.name}</span>
                    <small>{proxyMeta(proxy)}</small>
                  </button>
                ))}
              </div>
            </article>
          ))}
        </div>
      )}
    </section>
  );
}

function AutoStableView(props: {
  activeGroup?: ProxyGroupView;
  autoStable: AutoStableStatus;
  autoStableGroup: string;
  autoStableGroups: ProxyGroupView[];
  busy: boolean;
  healthBusy: boolean;
  healthRows: HealthRow[];
  healthSummary: ReturnType<typeof summarizeHealth>;
  refreshHealth(): Promise<void>;
  runTick(): void;
  setAutoStableGroup(value: string): void;
  toggle(enabled: boolean): void;
}) {
  return (
    <div className="view-stack">
      <section className="panel auto-hero">
        <div>
          <span className={props.autoStable.enabled ? "state-chip good" : "state-chip"}>{props.autoStable.enabled ? "自动选择已开启" : "自动选择未开启"}</span>
          <h3>稳定优先的节点选择</h3>
          <p>按延迟与失败率评分，配合冷却机制减少反复切换。</p>
        </div>
        <label className="switch-card">
          <span>启用自动选择</span>
          <input
            checked={props.autoStable.enabled}
            disabled={props.busy || !props.autoStable.available}
            onChange={(event) => props.toggle(event.target.checked)}
            type="checkbox"
          />
        </label>
      </section>

      <section className="quick-grid">
        <StatusTile label="最佳节点" value={props.healthSummary.bestNode || "暂无"} tone={props.healthSummary.bestNode ? "good" : "muted"} />
        <StatusTile label="健康节点" value={`${props.healthSummary.healthy}/${props.healthSummary.total}`} tone={props.healthSummary.healthy > 0 ? "good" : "muted"} />
        <StatusTile label="平均延迟" value={props.healthSummary.averageLatency ? `${props.healthSummary.averageLatency} ms` : "暂无"} tone="muted" />
        <StatusTile label="失败次数" value={`${props.healthSummary.failures}`} tone={props.healthSummary.failures > 0 ? "warn" : "good"} />
      </section>

      <section className="panel full-panel">
        <PanelTitle title="健康检测" meta={props.autoStable.running || props.healthBusy ? "检测中" : props.autoStable.lastTickAt ? formatRelativeTime(props.autoStable.lastTickAt) : "未检测"} />
        <div className="filter-bar">
          <select
            value={props.autoStableGroup}
            onChange={(event) => props.setAutoStableGroup(event.target.value)}
            disabled={props.busy || props.autoStableGroups.length === 0}
          >
            {props.autoStableGroups.length === 0 ? (
              <option value="AUTO-STABLE">AUTO-STABLE</option>
            ) : (
              props.autoStableGroups.map((group) => (
                <option key={group.name} value={group.name}>{group.name}</option>
              ))
            )}
          </select>
          <button className="ghost-button" disabled={props.busy || props.healthBusy || !props.autoStable.available} onClick={props.refreshHealth} type="button">刷新检测</button>
          <button className="primary-button" disabled={props.busy || props.healthBusy || !props.autoStable.available || !props.autoStable.enabled} onClick={props.runTick} type="button">立即选择</button>
        </div>

        {props.activeGroup ? (
          <div className="active-group-strip">
            <span>当前分组</span>
            <strong>{props.activeGroup.name}</strong>
            <small>{props.activeGroup.selected || "未选择"}</small>
          </div>
        ) : null}

        {props.healthRows.length === 0 ? (
          <EmptyState text={props.autoStable.available ? "暂无健康检测数据" : "加载订阅后会生成自动稳定分组"} />
        ) : (
          <div className="health-list">
            {props.healthRows.map((row) => (
              <div className="health-item" key={`${row.groupName}-${row.node.name}`}>
                <div>
                  <strong>{row.node.name}</strong>
                  <span>{row.groupName}</span>
                </div>
                <Metric label="评分" value={formatScore(row.node)} />
                <Metric label="延迟" value={formatLatency(row.node)} />
                <Metric label="失败率" value={formatFailureRate(row.node)} />
                <Metric label="检测时间" value={formatRelativeTime(row.node.lastCheckedAt)} />
              </div>
            ))}
          </div>
        )}
      </section>
    </div>
  );
}

function LogsView(props: {
  filteredLogs: LogLine[];
  logLevel: string;
  logLevels: string[];
  logQuery: string;
  logs: LogLine[];
  setLogLevel(value: string): void;
  setLogQuery(value: string): void;
}) {
  return (
    <section className="panel full-panel">
      <PanelTitle title="日志" meta={`${props.filteredLogs.length}/${props.logs.length} 条`} />
      <div className="filter-bar">
        <select value={props.logLevel} onChange={(event) => props.setLogLevel(event.target.value)}>
          <option value="all">全部级别</option>
          {props.logLevels.map((level) => (
            <option key={level} value={level}>{translateLogLevel(level)}</option>
          ))}
        </select>
        <input
          value={props.logQuery}
          onChange={(event) => props.setLogQuery(event.target.value)}
          placeholder="筛选日志内容"
          spellCheck={false}
        />
        <button
          className="ghost-button"
          disabled={props.logLevel === "all" && props.logQuery.length === 0}
          onClick={() => {
            props.setLogLevel("all");
            props.setLogQuery("");
          }}
          type="button"
        >
          清空
        </button>
      </div>
      <div className="log-list">
        {props.logs.length === 0 ? (
          <EmptyState text="暂无日志" />
        ) : props.filteredLogs.length === 0 ? (
          <EmptyState text="没有匹配的日志" />
        ) : (
          props.filteredLogs.map((line, index) => (
            <div className={`log-line ${logTone(line.level)}`} key={`${line.time}-${index}`}>
              <time>{formatClock(line.time)}</time>
              <span>{translateLogLevel(line.level)}</span>
              <p>{line.message}</p>
            </div>
          ))
        )}
      </div>
    </section>
  );
}

function SettingsView(props: {
  autoStable: AutoStableStatus;
  busy: boolean;
  status: AppStatus;
  subscriptionUrl: string;
  setSubscriptionUrl(value: string): void;
  submitSubscription(event: FormEvent<HTMLFormElement>): void;
  toggleAuto(enabled: boolean): void;
  toggleSystemProxy(enabled: boolean): void;
}) {
  return (
    <div className="settings-layout">
      <section className="panel">
        <PanelTitle title="订阅" meta={props.status.activeProfileName || "未加载"} />
        <form className="subscription-form" onSubmit={props.submitSubscription}>
          <input
            value={props.subscriptionUrl}
            onChange={(event) => props.setSubscriptionUrl(event.target.value)}
            placeholder="输入订阅地址"
            spellCheck={false}
          />
          <button className="primary-button" disabled={props.busy} type="submit">加载订阅</button>
        </form>
      </section>

      <section className="panel">
        <PanelTitle title="开关" meta="本机代理" />
        <div className="toggle-list">
          <label className="switch-card">
            <span>Windows 系统代理</span>
            <input
              checked={props.status.systemProxyEnabled}
              disabled={props.busy}
              onChange={(event) => props.toggleSystemProxy(event.target.checked)}
              type="checkbox"
            />
          </label>
          <label className="switch-card">
            <span>自动稳定选择</span>
            <input
              checked={props.autoStable.enabled}
              disabled={props.busy || !props.autoStable.available}
              onChange={(event) => props.toggleAuto(event.target.checked)}
              type="checkbox"
            />
          </label>
        </div>
      </section>
    </div>
  );
}

type HealthRow = {
  groupName: string;
  node: AutoStableNodeHealth;
};

function PanelTitle(props: { title: string; meta: string }) {
  return (
    <div className="panel-title">
      <h2>{props.title}</h2>
      <span>{props.meta}</span>
    </div>
  );
}

function StatusTile(props: { label: string; value: string; tone: "good" | "muted" | "warn" }) {
  return (
    <article className={`status-tile ${props.tone}`}>
      <span>{props.label}</span>
      <strong title={props.value}>{props.value}</strong>
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

function EmptyState(props: { text: string }) {
  return <div className="empty-state">{props.text}</div>;
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
      (group.selected || "").toLowerCase().includes(normalized) ||
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
  return Number.isFinite(score) ? score.toFixed(0) : "暂无";
}

function formatLatency(row: AutoStableNodeHealth): string {
  return typeof row.latencyMs === "number" && row.latencyMs > 0 ? `${row.latencyMs} ms` : "暂无";
}

function formatFailureRate(row: AutoStableNodeHealth): string {
  const total = row.totalChecks ?? ((row.successCount || 0) + (row.failureCount || 0));
  if (!total) {
    return "暂无";
  }
  const rate = row.failureRate ?? ((row.failureCount || 0) / total);
  return `${(rate * 100).toFixed(0)}%`;
}

function formatRelativeTime(value?: string): string {
  if (!value) {
    return "暂无";
  }
  const time = new Date(value).getTime();
  if (!Number.isFinite(time)) {
    return "暂无";
  }
  const seconds = Math.max(0, Math.round((Date.now() - time) / 1000));
  if (seconds < 60) {
    return `${seconds} 秒前`;
  }
  const minutes = Math.round(seconds / 60);
  if (minutes < 60) {
    return `${minutes} 分钟前`;
  }
  const hours = Math.round(minutes / 60);
  return `${hours} 小时前`;
}

function formatClock(value: string): string {
  const time = new Date(value);
  if (Number.isNaN(time.getTime())) {
    return "暂无";
  }
  return time.toLocaleTimeString("zh-CN", { hour12: false });
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

function translateGroupType(type: string): string {
  const normalized = type.toLowerCase();
  if (normalized.includes("auto-stable")) {
    return "自动稳定";
  }
  if (normalized.includes("url-test")) {
    return "延迟测试";
  }
  if (normalized.includes("fallback")) {
    return "故障回退";
  }
  if (normalized.includes("select")) {
    return "手动选择";
  }
  if (normalized.includes("auto")) {
    return "自动";
  }
  return type || "分组";
}

function proxyMeta(proxy: { type?: string; latencyMs?: number; alive: boolean }): string {
  const parts = [proxy.type || "节点"];
  if (typeof proxy.latencyMs === "number" && proxy.latencyMs > 0) {
    parts.push(`${proxy.latencyMs} ms`);
  }
  if (!proxy.alive) {
    parts.push("不可用");
  }
  return parts.join(" · ");
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

function translateLogLevel(level: string): string {
  const normalized = level.toLowerCase();
  if (normalized.includes("fatal")) {
    return "致命";
  }
  if (normalized.includes("error")) {
    return "错误";
  }
  if (normalized.includes("warn")) {
    return "警告";
  }
  if (normalized.includes("debug")) {
    return "调试";
  }
  if (normalized.includes("trace")) {
    return "追踪";
  }
  return level || "信息";
}

function normalizeAppStatus(status: AppStatus | null | undefined): AppStatus {
  return { ...emptyStatus, ...(status ?? {}) };
}

function normalizeConnection(status: ConnectionStatus | null | undefined): ConnectionStatus {
  return { ...emptyConnection, ...(status ?? {}) };
}

function normalizeGroups(groups: ProxyGroupView[] | null | undefined): ProxyGroupView[] {
  return (groups ?? [])
    .filter((group): group is ProxyGroupView => Boolean(group))
    .map((group) => ({
      ...group,
      name: group.name || "未命名分组",
      type: group.type || "select",
      selected: group.selected || "",
      proxies: (group.proxies ?? [])
        .filter((proxy): proxy is ProxyGroupView["proxies"][number] => Boolean(proxy))
        .map((proxy) => ({
          ...proxy,
          name: proxy.name || "未命名节点",
          type: proxy.type || "",
          alive: Boolean(proxy.alive)
        }))
    }));
}

function normalizeLogs(logs: LogLine[] | null | undefined): LogLine[] {
  return (logs ?? [])
    .filter((line): line is LogLine => Boolean(line))
    .map((line) => ({
      ...line,
      time: line.time || "",
      level: line.level || "info",
      message: line.message || ""
    }));
}

function normalizeAutoStable(status: AutoStableStatus | null | undefined): AutoStableStatus {
  const next = { ...emptyAutoStable, ...(status ?? {}) };
  return {
    ...next,
    health: (next.health ?? [])
      .filter((group): group is AutoStableGroupHealth => Boolean(group))
      .map((group) => ({
        ...group,
        name: group.name || "未命名分组",
        type: group.type || "auto-stable",
        selected: group.selected || "",
        proxies: (group.proxies ?? [])
          .filter((node): node is AutoStableNodeHealth => Boolean(node))
          .map((node) => ({
            ...node,
            name: node.name || "未命名节点",
            type: node.type || "",
            alive: Boolean(node.alive)
          }))
      }))
  };
}

function toMessage(err: unknown): string {
  return err instanceof Error ? err.message : String(err);
}
