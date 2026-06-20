import { FormEvent, useEffect, useMemo, useState } from "react";
import {
  AppStatus,
  AutoStableGroupHealth,
  AutoStableNodeHealth,
  AutoStableStatus,
  ConnectionStatus,
  LogLine,
  ProxyGroupView,
  ProxyView,
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

type ViewId = "connection" | "subscription" | "nodes" | "groups" | "auto" | "diagnostics";

const navItems: Array<{ id: ViewId; label: string; description: string }> = [
  { id: "connection", label: "连接", description: "启动、断开、看当前出口" },
  { id: "subscription", label: "订阅", description: "加载配置来源" },
  { id: "nodes", label: "节点", description: "查看节点状态" },
  { id: "groups", label: "代理组", description: "手动切换分组" },
  { id: "auto", label: "自动选择", description: "稳定优先策略" },
  { id: "diagnostics", label: "诊断", description: "定位连接问题" }
];

export default function App() {
  const [status, setStatus] = useState<AppStatus>(emptyStatus);
  const [autoStable, setAutoStable] = useState<AutoStableStatus>(emptyAutoStable);
  const [connection, setConnection] = useState<ConnectionStatus>(emptyConnection);
  const [groups, setGroups] = useState<ProxyGroupView[]>([]);
  const [logs, setLogs] = useState<LogLine[]>([]);
  const [subscriptionUrl, setSubscriptionUrl] = useState("");
  const [activeView, setActiveView] = useState<ViewId>("connection");
  const [selectedGroupName, setSelectedGroupName] = useState("");
  const [autoStableGroupName, setAutoStableGroupName] = useState("AUTO-STABLE");
  const [nodeQuery, setNodeQuery] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const hasProfile = groups.length > 0 || Boolean(status.activeProfileName);
  const connected = status.coreRunning && status.systemProxyEnabled;

  async function refresh() {
    try {
      const [nextStatus, nextGroups, nextLogs, nextAutoStable, nextConnection] = await Promise.all([
        getStatus(),
        getProxyGroups(),
        getLogs(120),
        getAutoStableStatus(),
        getConnectionStatus()
      ]);
      const normalizedStatus = normalizeAppStatus(nextStatus);
      const normalizedGroups = normalizeGroups(nextGroups);
      const normalizedAutoStable = normalizeAutoStable(nextAutoStable);
      setStatus(normalizedStatus);
      setGroups(normalizedGroups);
      setLogs(normalizeLogs(nextLogs));
      setAutoStable(normalizedAutoStable);
      setConnection(normalizeConnection(nextConnection));
      setError(normalizedStatus.lastError || normalizedAutoStable.lastError || null);
      syncSelectedGroups(normalizedGroups, normalizedAutoStable);
    } catch (err) {
      setError(toMessage(err));
    }
  }

  function syncSelectedGroups(nextGroups: ProxyGroupView[], nextAutoStable: AutoStableStatus) {
    if (nextGroups.length > 0) {
      setSelectedGroupName((current) => current && nextGroups.some((group) => group.name === current) ? current : nextGroups[0].name);
    }
    const autoGroups = findAutoStableGroups(nextGroups);
    const healthGroups = nextAutoStable.health.map((group) => group.name);
    const candidates = autoGroups.map((group) => group.name).concat(healthGroups);
    if (candidates.length > 0) {
      setAutoStableGroupName((current) => current && candidates.includes(current) ? current : candidates[0]);
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

  async function connect() {
    if (!hasProfile) {
      setError("请先在“订阅”中加载节点订阅");
      setActiveView("subscription");
      return;
    }
    await run(async () => {
      if (!status.coreRunning) {
        await startCore();
      }
      await setSystemProxy(true);
    });
  }

  async function disconnect() {
    await run(async () => {
      if (status.systemProxyEnabled) {
        await setSystemProxy(false);
      }
      if (status.coreRunning) {
        await stopCore();
      }
    });
  }

  async function submitSubscription(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const url = subscriptionUrl.trim();
    if (!url) {
      setError("请输入订阅地址");
      return;
    }
    await run(async () => {
      await loadSubscription(url);
    });
    setActiveView("connection");
  }

  useEffect(() => {
    refresh();
    const id = window.setInterval(refresh, 5000);
    return () => window.clearInterval(id);
  }, []);

  const selectedGroup = useMemo(
    () => groups.find((group) => group.name === selectedGroupName) || groups[0],
    [groups, selectedGroupName]
  );
  const nodeRows = useMemo(() => filterNodeRows(flattenNodes(groups), nodeQuery), [groups, nodeQuery]);
  const autoGroups = useMemo(() => findAutoStableGroups(groups), [groups]);
  const healthRows = useMemo(
    () => flattenHealth(autoStable.health).sort((left, right) => healthScore(left.node) - healthScore(right.node)),
    [autoStable.health]
  );
  const healthSummary = useMemo(() => summarizeHealth(healthRows), [healthRows]);
  const recentErrorLogs = useMemo(() => logs.filter((line) => isErrorLevel(line.level)).slice(0, 8), [logs]);

  return (
    <main className="pc-shell">
      <aside className="pc-sidebar" aria-label="主导航">
        <div className="pc-brand">
          <strong>Proxy-Cat</strong>
          <span>Mihomo 代理客户端</span>
        </div>

        <nav className="pc-nav">
          {navItems.map((item) => (
            <button
              className={activeView === item.id ? "pc-nav-item active" : "pc-nav-item"}
              key={item.id}
              onClick={() => setActiveView(item.id)}
              type="button"
            >
              <strong>{item.label}</strong>
              <span>{item.description}</span>
            </button>
          ))}
        </nav>

        <div className="pc-sidebar-status">
          <StatusBadge tone={connected ? "good" : status.coreRunning ? "warn" : "neutral"}>
            {connected ? "已连接" : status.coreRunning ? "内核运行" : "未连接"}
          </StatusBadge>
          <span>{status.controllerAddress}</span>
        </div>
      </aside>

      <section className="pc-main">
        <header className="pc-topbar">
          <div>
            <p>{currentTitle(activeView)}</p>
            <h1>{currentDescription(activeView)}</h1>
          </div>
          <div className="pc-top-actions">
            <button className="secondary" disabled={busy} onClick={refresh} type="button">刷新</button>
            <button className="secondary" disabled={busy || !status.coreRunning} onClick={() => run(restartCore)} type="button">重启内核</button>
            <button
              className={connected ? "danger" : "primary"}
              disabled={busy}
              onClick={hasProfile ? (connected ? disconnect : connect) : () => setActiveView("subscription")}
              type="button"
            >
              {hasProfile ? (connected ? "断开连接" : "连接") : "添加订阅"}
            </button>
          </div>
        </header>

        {error ? <Alert tone="error" title="当前错误" message={error} /> : null}

        {activeView === "connection" ? (
          <ConnectionView
            autoStable={autoStable}
            busy={busy}
            connected={connected}
            connection={connection}
            groups={groups}
            hasProfile={hasProfile}
            healthSummary={healthSummary}
            onConnect={connect}
            onDisconnect={disconnect}
            onGoSubscription={() => setActiveView("subscription")}
            status={status}
          />
        ) : null}

        {activeView === "subscription" ? (
          <SubscriptionView
            busy={busy}
            groups={groups}
            onSubmit={submitSubscription}
            setSubscriptionUrl={setSubscriptionUrl}
            status={status}
            subscriptionUrl={subscriptionUrl}
          />
        ) : null}

        {activeView === "nodes" ? (
          <NodesView nodeQuery={nodeQuery} nodeRows={nodeRows} setNodeQuery={setNodeQuery} />
        ) : null}

        {activeView === "groups" ? (
          <GroupsView
            busy={busy}
            groups={groups}
            onSelect={(groupName, proxyName) => run(() => selectProxy(groupName, proxyName))}
            selectedGroup={selectedGroup}
            selectedGroupName={selectedGroupName}
            setSelectedGroupName={setSelectedGroupName}
          />
        ) : null}

        {activeView === "auto" ? (
          <AutoStableView
            autoGroups={autoGroups}
            autoStable={autoStable}
            busy={busy}
            healthRows={healthRows}
            healthSummary={healthSummary}
            selectedGroupName={autoStableGroupName}
            setSelectedGroupName={setAutoStableGroupName}
            toggleAuto={(enabled) => run(() => setAutoStableEnabled(enabled))}
            runTick={() => run(async () => {
              await runAutoStableTick();
            })}
          />
        ) : null}

        {activeView === "diagnostics" ? (
          <DiagnosticsView
            autoStable={autoStable}
            connection={connection}
            error={error}
            groups={groups}
            logs={logs}
            recentErrorLogs={recentErrorLogs}
            status={status}
          />
        ) : null}
      </section>
    </main>
  );
}

function ConnectionView(props: {
  autoStable: AutoStableStatus;
  busy: boolean;
  connected: boolean;
  connection: ConnectionStatus;
  groups: ProxyGroupView[];
  hasProfile: boolean;
  healthSummary: ReturnType<typeof summarizeHealth>;
  onConnect(): void;
  onDisconnect(): void;
  onGoSubscription(): void;
  status: AppStatus;
}) {
  const selected = selectedNodes(props.groups);
  const currentExit = selected[0] || "暂无";

  return (
    <div className="pc-stack">
      <section className="pc-connection-panel">
        <div>
          <StatusBadge tone={props.connected ? "good" : props.status.coreRunning ? "warn" : "neutral"}>
            {props.connected ? "已连接" : props.status.coreRunning ? "内核运行，系统代理未开启" : "未连接"}
          </StatusBadge>
          <h2>{props.hasProfile ? (props.connected ? "代理已经接管系统流量" : "连接后开始代理") : "加载订阅后开始代理"}</h2>
          <p>
            {props.hasProfile
              ? `当前订阅：${props.status.activeProfileName || "默认订阅"}`
              : "还没有订阅，请先加载节点订阅。"}
          </p>
        </div>
        <div className="pc-primary-action">
          {props.hasProfile ? (
            <button className={props.connected ? "danger large" : "primary large"} disabled={props.busy} onClick={props.connected ? props.onDisconnect : props.onConnect} type="button">
              {props.connected ? "断开连接" : "连接"}
            </button>
          ) : (
            <button className="primary large" disabled={props.busy} onClick={props.onGoSubscription} type="button">添加订阅</button>
          )}
        </div>
      </section>

      <section className="pc-section">
        <SectionTitle title="当前状态" meta="连接视图只保留必要信息" />
        <div className="pc-status-grid">
          <StatusMetric label="Mihomo 内核" value={props.status.coreRunning ? "运行中" : "未启动"} tone={props.status.coreRunning ? "good" : "neutral"} />
          <StatusMetric label="Windows 系统代理" value={props.status.systemProxyEnabled ? "已开启" : "未开启"} tone={props.status.systemProxyEnabled ? "good" : "neutral"} />
          <StatusMetric label="当前出口" value={currentExit} tone={currentExit === "暂无" ? "neutral" : "good"} />
          <StatusMetric label="连接数" value={`${props.connection.connectionCount}`} tone={props.connection.connectionCount > 0 ? "good" : "neutral"} />
        </div>
      </section>

      <section className="pc-section">
        <SectionTitle title="自动选择" meta={props.autoStable.available ? "可用" : "未就绪"} />
        <div className="pc-two-column">
          <StatusMetric label="稳定优先" value={props.autoStable.enabled ? "已开启" : "未开启"} tone={props.autoStable.enabled ? "good" : "neutral"} />
          <StatusMetric label="最佳节点" value={props.healthSummary.bestNode || "暂无检测"} tone={props.healthSummary.bestNode ? "good" : "neutral"} />
        </div>
      </section>

      <section className="pc-section">
        <SectionTitle title="当前代理组选择" meta={`${selected.length} 个分组`} />
        {selected.length === 0 ? (
          <EmptyState title="暂无代理组" action="加载订阅后会显示当前出口" />
        ) : (
          <div className="pc-simple-list">
            {selected.map((item) => <span key={item}>{item}</span>)}
          </div>
        )}
      </section>
    </div>
  );
}

function SubscriptionView(props: {
  busy: boolean;
  groups: ProxyGroupView[];
  onSubmit(event: FormEvent<HTMLFormElement>): void;
  setSubscriptionUrl(value: string): void;
  status: AppStatus;
  subscriptionUrl: string;
}) {
  const nodeCount = flattenNodes(props.groups).length;
  return (
    <div className="pc-stack">
      <section className="pc-section">
        <SectionTitle title="订阅地址" meta={props.status.activeProfileName || "未加载"} />
        <form className="pc-form" onSubmit={props.onSubmit}>
          <label>
            <span>节点订阅 URL</span>
            <input
              autoComplete="off"
              onChange={(event) => props.setSubscriptionUrl(event.target.value)}
              placeholder="https://example.com/sub"
              spellCheck={false}
              type="url"
              value={props.subscriptionUrl}
            />
          </label>
          <button className="primary" disabled={props.busy} type="submit">加载订阅</button>
        </form>
      </section>

      <section className="pc-section">
        <SectionTitle title="解析结果" meta="用于判断配置是否可启动" />
        <div className="pc-status-grid">
          <StatusMetric label="配置名称" value={props.status.activeProfileName || "暂无"} tone={props.status.activeProfileName ? "good" : "neutral"} />
          <StatusMetric label="代理组" value={`${props.groups.length}`} tone={props.groups.length > 0 ? "good" : "neutral"} />
          <StatusMetric label="节点引用" value={`${nodeCount}`} tone={nodeCount > 0 ? "good" : "neutral"} />
          <StatusMetric label="控制端口" value={props.status.controllerAddress} tone="neutral" />
        </div>
      </section>
    </div>
  );
}

function NodesView(props: {
  nodeQuery: string;
  nodeRows: NodeRow[];
  setNodeQuery(value: string): void;
}) {
  return (
    <section className="pc-section fill">
      <SectionTitle title="节点" meta={`${props.nodeRows.length} 条`} />
      <div className="pc-toolbar">
        <label className="pc-search">
          <span>搜索节点</span>
          <input
            onChange={(event) => props.setNodeQuery(event.target.value)}
            placeholder="输入节点名、类型或分组"
            spellCheck={false}
            value={props.nodeQuery}
          />
        </label>
      </div>
      {props.nodeRows.length === 0 ? (
        <EmptyState title="暂无节点" action="加载订阅后会显示节点列表" />
      ) : (
        <div className="pc-table-wrap">
          <table className="pc-table">
            <thead>
              <tr>
                <th>节点</th>
                <th>分组</th>
                <th>类型</th>
                <th>延迟</th>
                <th>状态</th>
              </tr>
            </thead>
            <tbody>
              {props.nodeRows.map((row) => (
                <tr key={`${row.groupName}-${row.proxy.name}`}>
                  <td>{row.proxy.name}</td>
                  <td>{row.groupName}</td>
                  <td>{row.proxy.type || "未知"}</td>
                  <td>{formatLatency(row.proxy)}</td>
                  <td><StatusBadge tone={row.proxy.alive ? "good" : "bad"}>{row.proxy.alive ? "可用" : "不可用"}</StatusBadge></td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </section>
  );
}

function GroupsView(props: {
  busy: boolean;
  groups: ProxyGroupView[];
  onSelect(groupName: string, proxyName: string): void;
  selectedGroup?: ProxyGroupView;
  selectedGroupName: string;
  setSelectedGroupName(value: string): void;
}) {
  return (
    <div className="pc-master-detail">
      <section className="pc-section pc-master">
        <SectionTitle title="代理组" meta={`${props.groups.length} 个`} />
        {props.groups.length === 0 ? (
          <EmptyState title="暂无代理组" action="加载订阅后会显示分组" />
        ) : (
          <div className="pc-group-list">
            {props.groups.map((group) => (
              <button
                className={group.name === props.selectedGroupName ? "pc-group-row active" : "pc-group-row"}
                key={group.name}
                onClick={() => props.setSelectedGroupName(group.name)}
                type="button"
              >
                <strong>{group.name}</strong>
                <span>{translateGroupType(group.type)} · {group.selected || "未选择"}</span>
              </button>
            ))}
          </div>
        )}
      </section>

      <section className="pc-section pc-detail">
        <SectionTitle title={props.selectedGroup?.name || "选择一个代理组"} meta={props.selectedGroup ? translateGroupType(props.selectedGroup.type) : "无"} />
        {!props.selectedGroup ? (
          <EmptyState title="暂无节点" action="从左侧选择代理组" />
        ) : (
          <div className="pc-table-wrap">
            <table className="pc-table">
              <thead>
                <tr>
                  <th>节点</th>
                  <th>类型</th>
                  <th>状态</th>
                  <th>操作</th>
                </tr>
              </thead>
              <tbody>
                {props.selectedGroup.proxies.map((proxy) => (
                  <tr key={`${props.selectedGroup?.name}-${proxy.name}`}>
                    <td>
                      <div className="pc-node-name">
                        <strong>{proxy.name}</strong>
                        {proxy.name === props.selectedGroup?.selected ? <StatusBadge tone="good">当前</StatusBadge> : null}
                      </div>
                    </td>
                    <td>{proxy.type || "未知"}</td>
                    <td><StatusBadge tone={proxy.alive ? "good" : "bad"}>{proxy.alive ? "可用" : "不可用"}</StatusBadge></td>
                    <td>
                      <button
                        className="secondary compact"
                        disabled={props.busy || proxy.name === props.selectedGroup?.selected}
                        onClick={() => props.selectedGroup && props.onSelect(props.selectedGroup.name, proxy.name)}
                        type="button"
                      >
                        选择
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>
    </div>
  );
}

function AutoStableView(props: {
  autoGroups: ProxyGroupView[];
  autoStable: AutoStableStatus;
  busy: boolean;
  healthRows: HealthRow[];
  healthSummary: ReturnType<typeof summarizeHealth>;
  selectedGroupName: string;
  setSelectedGroupName(value: string): void;
  toggleAuto(enabled: boolean): void;
  runTick(): void;
}) {
  const availableGroups = props.autoGroups.map((group) => group.name);
  const healthGroups = Array.from(new Set(props.healthRows.map((row) => row.groupName)));
  const groupOptions = Array.from(new Set(availableGroups.concat(healthGroups)));
  const rows = props.selectedGroupName
    ? props.healthRows.filter((row) => row.groupName === props.selectedGroupName)
    : props.healthRows;

  return (
    <div className="pc-stack">
      <section className="pc-section">
        <SectionTitle title="稳定优先" meta={props.autoStable.available ? "可用" : "订阅加载后可用"} />
        <div className="pc-action-row">
          <label className="pc-toggle">
            <input
              checked={props.autoStable.enabled}
              disabled={props.busy || !props.autoStable.available}
              onChange={(event) => props.toggleAuto(event.target.checked)}
              type="checkbox"
            />
            <span>启用自动稳定选择</span>
          </label>
          <button className="primary" disabled={props.busy || !props.autoStable.enabled || !props.autoStable.available} onClick={props.runTick} type="button">
            立即选择一次
          </button>
        </div>
      </section>

      <section className="pc-section">
        <SectionTitle title="评分摘要" meta="score = latency + failure_rate" />
        <div className="pc-status-grid">
          <StatusMetric label="最佳节点" value={props.healthSummary.bestNode || "暂无"} tone={props.healthSummary.bestNode ? "good" : "neutral"} />
          <StatusMetric label="健康节点" value={`${props.healthSummary.healthy}/${props.healthSummary.total}`} tone={props.healthSummary.healthy > 0 ? "good" : "neutral"} />
          <StatusMetric label="平均延迟" value={props.healthSummary.averageLatency ? `${props.healthSummary.averageLatency} ms` : "暂无"} tone="neutral" />
          <StatusMetric label="最近动作" value={props.autoStable.lastAction || "暂无"} tone="neutral" />
        </div>
      </section>

      <section className="pc-section fill">
        <SectionTitle title="健康检测" meta={props.autoStable.lastTickAt ? formatRelativeTime(props.autoStable.lastTickAt) : "未检测"} />
        <div className="pc-toolbar">
          <label>
            <span>代理组</span>
            <select
              disabled={groupOptions.length === 0}
              onChange={(event) => props.setSelectedGroupName(event.target.value)}
              value={props.selectedGroupName}
            >
              {groupOptions.length === 0 ? <option value="AUTO-STABLE">AUTO-STABLE</option> : null}
              {groupOptions.map((name) => <option key={name} value={name}>{name}</option>)}
            </select>
          </label>
        </div>
        {rows.length === 0 ? (
          <EmptyState title="暂无检测数据" action="开启自动选择后会显示节点健康数据" />
        ) : (
          <div className="pc-table-wrap">
            <table className="pc-table">
              <thead>
                <tr>
                  <th>节点</th>
                  <th>评分</th>
                  <th>延迟</th>
                  <th>失败率</th>
                  <th>检测时间</th>
                  <th>状态</th>
                </tr>
              </thead>
              <tbody>
                {rows.map((row) => (
                  <tr key={`${row.groupName}-${row.node.name}`}>
                    <td>{row.node.name}</td>
                    <td>{formatScore(row.node)}</td>
                    <td>{formatHealthLatency(row.node)}</td>
                    <td>{formatFailureRate(row.node)}</td>
                    <td>{formatRelativeTime(row.node.lastCheckedAt)}</td>
                    <td><StatusBadge tone={row.node.alive ? "good" : "bad"}>{row.node.alive ? "可用" : "失败"}</StatusBadge></td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>
    </div>
  );
}

function DiagnosticsView(props: {
  autoStable: AutoStableStatus;
  connection: ConnectionStatus;
  error: string | null;
  groups: ProxyGroupView[];
  logs: LogLine[];
  recentErrorLogs: LogLine[];
  status: AppStatus;
}) {
  const checks = [
    {
      name: "订阅配置",
      ok: props.groups.length > 0,
      detail: props.groups.length > 0 ? `${props.groups.length} 个代理组已加载` : "未加载订阅"
    },
    {
      name: "Mihomo 内核",
      ok: props.status.coreRunning,
      detail: props.status.coreRunning ? "正在运行" : "未启动"
    },
    {
      name: "系统代理",
      ok: props.status.systemProxyEnabled,
      detail: props.status.systemProxyEnabled ? "Windows 代理已开启" : "Windows 代理未开启"
    },
    {
      name: "控制端口",
      ok: Boolean(props.status.controllerAddress),
      detail: props.status.controllerAddress || "未配置"
    },
    {
      name: "自动选择",
      ok: props.autoStable.available,
      detail: props.autoStable.available ? (props.autoStable.enabled ? "已开启" : "可用但未开启") : "订阅加载后可用"
    }
  ];

  return (
    <div className="pc-stack">
      <section className="pc-section">
        <SectionTitle title="诊断检查" meta={props.error ? "存在错误" : "暂无错误"} />
        <div className="pc-check-list">
          {checks.map((check) => (
            <div className="pc-check-row" key={check.name}>
              <StatusBadge tone={check.ok ? "good" : "warn"}>{check.ok ? "正常" : "待处理"}</StatusBadge>
              <strong>{check.name}</strong>
              <span>{check.detail}</span>
            </div>
          ))}
        </div>
      </section>

      <section className="pc-section">
        <SectionTitle title="运行数据" meta="Mihomo /connections" />
        <div className="pc-status-grid">
          <StatusMetric label="连接数" value={`${props.connection.connectionCount}`} tone="neutral" />
          <StatusMetric label="下载" value={formatBytes(props.connection.downloadTotal)} tone="neutral" />
          <StatusMetric label="上传" value={formatBytes(props.connection.uploadTotal)} tone="neutral" />
          <StatusMetric label="日志数" value={`${props.logs.length}`} tone="neutral" />
        </div>
      </section>

      <section className="pc-section fill">
        <SectionTitle title="最近错误日志" meta={`${props.recentErrorLogs.length} 条`} />
        {props.recentErrorLogs.length === 0 ? (
          <EmptyState title="暂无错误日志" action="连接异常时这里会显示最近错误" />
        ) : (
          <LogList logs={props.recentErrorLogs} />
        )}
      </section>

      <section className="pc-section fill">
        <SectionTitle title="最近日志" meta={`${props.logs.length} 条`} />
        {props.logs.length === 0 ? <EmptyState title="暂无日志" action="启动内核或加载订阅后会产生日志" /> : <LogList logs={props.logs.slice(0, 40)} />}
      </section>
    </div>
  );
}

function LogList(props: { logs: LogLine[] }) {
  return (
    <div className="pc-log-list">
      {props.logs.map((line, index) => (
        <div className="pc-log-row" key={`${line.time}-${index}`}>
          <time>{formatClock(line.time)}</time>
          <StatusBadge tone={isErrorLevel(line.level) ? "bad" : isWarnLevel(line.level) ? "warn" : "neutral"}>
            {translateLogLevel(line.level)}
          </StatusBadge>
          <span>{line.message}</span>
        </div>
      ))}
    </div>
  );
}

function SectionTitle(props: { title: string; meta?: string }) {
  return (
    <div className="pc-section-title">
      <h2>{props.title}</h2>
      {props.meta ? <span>{props.meta}</span> : null}
    </div>
  );
}

function Alert(props: { tone: "error" | "info"; title: string; message: string }) {
  return (
    <section className={`pc-alert ${props.tone}`}>
      <strong>{props.title}</strong>
      <span>{props.message}</span>
    </section>
  );
}

function StatusMetric(props: { label: string; value: string; tone: BadgeTone }) {
  return (
    <div className="pc-metric">
      <span>{props.label}</span>
      <strong title={props.value}>{props.value}</strong>
      {props.tone === "neutral" ? null : <StatusBadge tone={props.tone}>{badgeText(props.tone)}</StatusBadge>}
    </div>
  );
}

type BadgeTone = "good" | "warn" | "bad" | "neutral";

function StatusBadge(props: { tone: BadgeTone; children: string }) {
  return <span className={`pc-badge ${props.tone}`}>{props.children}</span>;
}

function EmptyState(props: { title: string; action: string }) {
  return (
    <div className="pc-empty">
      <strong>{props.title}</strong>
      <span>{props.action}</span>
    </div>
  );
}

type NodeRow = {
  groupName: string;
  proxy: ProxyView;
};

type HealthRow = {
  groupName: string;
  node: AutoStableNodeHealth;
};

function flattenNodes(groups: ProxyGroupView[]): NodeRow[] {
  return groups.flatMap((group) => group.proxies.map((proxy) => ({ groupName: group.name, proxy })));
}

function filterNodeRows(rows: NodeRow[], query: string): NodeRow[] {
  const normalized = query.trim().toLowerCase();
  if (!normalized) {
    return rows;
  }
  return rows.filter((row) =>
    `${row.groupName} ${row.proxy.name} ${row.proxy.type || ""}`.toLowerCase().includes(normalized)
  );
}

function selectedNodes(groups: ProxyGroupView[]): string[] {
  return groups.filter((group) => group.selected).map((group) => `${group.name}：${group.selected}`);
}

function findAutoStableGroups(groups: ProxyGroupView[]): ProxyGroupView[] {
  return groups.filter((group) => {
    const name = group.name.toLowerCase();
    const type = group.type.toLowerCase();
    return type.includes("auto") || name.includes("auto") || name.includes("stable") || name.includes("自动");
  });
}

function flattenHealth(groups: AutoStableGroupHealth[]): HealthRow[] {
  return groups.flatMap((group) =>
    group.proxies.map((node) => ({
      groupName: group.name,
      node
    }))
  );
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

function formatLatency(row: ProxyView): string {
  return typeof row.latencyMs === "number" && row.latencyMs > 0 ? `${row.latencyMs} ms` : "暂无";
}

function formatHealthLatency(row: AutoStableNodeHealth): string {
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
  if (!Number.isFinite(time) || time <= 0) {
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

function isErrorLevel(level: string): boolean {
  const normalized = level.toLowerCase();
  return normalized.includes("error") || normalized.includes("fatal");
}

function isWarnLevel(level: string): boolean {
  return level.toLowerCase().includes("warn");
}

function badgeText(tone: BadgeTone): string {
  switch (tone) {
    case "good":
      return "正常";
    case "warn":
      return "注意";
    case "bad":
      return "异常";
    default:
      return "状态";
  }
}

function currentTitle(view: ViewId): string {
  return navItems.find((item) => item.id === view)?.label || "连接";
}

function currentDescription(view: ViewId): string {
  switch (view) {
    case "connection":
      return "先确认能不能代理，再处理其他事情";
    case "subscription":
      return "加载订阅并生成 Mihomo 配置";
    case "nodes":
      return "检查节点是否可用";
    case "groups":
      return "按代理组选择出口节点";
    case "auto":
      return "用延迟和失败率选择更稳定的节点";
    case "diagnostics":
      return "连接失败时从这里定位问题";
    default:
      return "";
  }
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
