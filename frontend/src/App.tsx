import { useState } from "react";
import type { ViewId } from "./types";
import { useAppState } from "./hooks/useAppState";
import { useActionRunner } from "./hooks/useActionRunner";
import { usePolling } from "./hooks/usePolling";
import {
  startCore, stopCore, restartCore,
  setSystemProxy, loadSubscription,
  selectProxy, setAutoStableEnabled,
  runAutoStableTick, selectAutoStableProxy,
  getAutoStableStatus,
} from "./api/client";
import { OverviewView } from "./components/OverviewView";
import { ProxyViewPanel } from "./components/ProxyView";
import { AutoStableView } from "./components/AutoStableView";
import { LogsView } from "./components/LogsView";
import { SettingsView } from "./components/SettingsView";
import { ContextPanel } from "./components/ContextPanel";
import { MatrixView } from "./components/MatrixView";
import { RuleEditorView } from "./components/RuleEditorView";
import { PolicyView } from "./components/PolicyView";

const NAV_ITEMS: { id: ViewId; mark: string; label: string }[] = [
  { id: "overview", mark: "览", label: "概览" },
  { id: "proxies", mark: "代", label: "代理" },
  { id: "auto", mark: "稳", label: "自动选择" },
  { id: "matrix", mark: "阵", label: "矩阵" },
  { id: "rules", mark: "策", label: "策略" },
  { id: "logs", mark: "志", label: "日志" },
  { id: "settings", mark: "设", label: "设置" },
];

export default function App() {
  const { status, autoStable, connection, groups, logs, setLogs, refresh, refreshHealth } = useAppState();
  const { busy, error, run, clearError } = useActionRunner();
  const [activeView, setActiveView] = useState<ViewId>("overview");
  const [subscriptionUrl, setSubscriptionUrl] = useState("");
  const [healthBusy, setHealthBusy] = useState(false);

  // Poll every 5 seconds
  usePolling(refresh, 5000);

  const doRefresh = async () => { await refresh(); await refreshHealth(); };

  return (
    <div className="grid grid-cols-[236px_minmax(0,1fr)] min-h-screen max-[780px]:grid-cols-1">
      {/* Sidebar */}
      <aside className="sticky top-0 h-screen flex flex-col bg-[rgb(255_250_247/0.9)] backdrop-blur-[20px] px-4 py-5 border-r border-[rgb(112_76_65/0.08)] max-[780px]:static max-[780px]:h-auto">
        <div className="mb-6">
          <div className="w-12 h-12 rounded-2xl bg-gradient-to-br from-[#ffd4c8] to-[#f3a996] flex items-center justify-center text-white font-extrabold text-lg mb-2">PC</div>
          <h1 className="text-lg font-extrabold text-brand-900">Proxy-Cat</h1>
          <p className="text-xs text-brand-500">更稳定的 Mihomo 客户端</p>
        </div>

        <nav className="flex-1 space-y-1">
          {NAV_ITEMS.map(item => (
            <button key={item.id}
              onClick={() => setActiveView(item.id)}
              className={`w-full flex items-center gap-3 px-3 py-2.5 rounded-2xl text-sm transition-colors ${
                activeView === item.id
                  ? "bg-[#ffe4dc] text-brand-900 font-bold"
                  : "text-brand-700 hover:bg-[#fff4ef]"
              }`}>
              <span className={`w-7 h-7 rounded-full flex items-center justify-center text-xs font-extrabold ${
                activeView === item.id
                  ? "bg-[#f3a793] text-white"
                  : "bg-[#fff4ef] text-brand-500"
              }`}>{item.mark}</span>
              {item.label}
            </button>
          ))}
        </nav>

        <div className="mt-auto pt-4 border-t border-[rgb(112_76_65/0.08)]">
          <span className={`inline-block px-3 py-1 rounded-full text-xs font-bold ${
            status.coreRunning ? "bg-[#dff1df] text-[#3d7145]" : "bg-[#ffe7df] text-[#8f321f]"
          }`}>
            {status.coreRunning ? "内核运行中" : "内核未启动"}
          </span>
          <div className="text-xs text-brand-400 mt-2 truncate">{status.controllerAddress}</div>
        </div>
      </aside>

      {/* Main workspace */}
      <main className="flex flex-col gap-4 p-5">
        {/* Topbar */}
        <header className="sticky top-0 z-10 flex items-center justify-between bg-[rgb(255_250_247/0.9)] backdrop-blur-[18px] rounded-2xl px-5 py-3 shadow-[0_12px_32px_rgb(124_76_62/0.06)]">
          <h2 className="text-[22px] font-normal text-brand-900">{status.activeProfileName || "未加载订阅"}</h2>
          <div className="flex gap-2">
            <button onClick={doRefresh} disabled={busy} className="rounded-xl bg-[#fff4ef] px-4 py-2 text-sm text-brand-700 hover:bg-[#ffe4dc] disabled:opacity-50">刷新</button>
            <button onClick={() => run(stopCore, doRefresh)} disabled={busy || !status.coreRunning} className="rounded-xl bg-[#fff4ef] px-4 py-2 text-sm text-brand-700 hover:bg-[#ffe4dc] disabled:opacity-50">停止</button>
            <button onClick={() => run(restartCore, doRefresh)} disabled={busy} className="rounded-xl bg-[#fff4ef] px-4 py-2 text-sm text-brand-700 hover:bg-[#ffe4dc] disabled:opacity-50">重启</button>
            <button onClick={() => run(startCore, doRefresh)} disabled={busy || status.coreRunning} className="rounded-xl bg-[#dc7f69] px-4 py-2 text-sm text-white hover:bg-[#cd705d] disabled:opacity-50">启动内核</button>
          </div>
        </header>

        {/* Error notice */}
        {error && (
          <div className="rounded-xl bg-[#ffe7df] text-[#8f321f] px-4 py-3 text-sm flex justify-between items-center">
            <span>{error}</span>
            <button onClick={clearError} className="font-bold ml-4">&times;</button>
          </div>
        )}

        {/* Content + context */}
        <div className="grid grid-cols-[minmax(0,1fr)_316px] gap-4 max-[1080px]:grid-cols-1">
          <div className="min-w-0">
            {activeView === "overview" && <OverviewView status={status} connection={connection} groups={groups} logs={logs} health={autoStable.health} />}
            {activeView === "proxies" && <ProxyViewPanel groups={groups} busy={busy} onSelect={(g, p) => run(() => selectProxy(g, p), doRefresh)} />}
            {activeView === "auto" && (
              <AutoStableView autoStable={autoStable} healthBusy={healthBusy}
                onTick={async () => { setHealthBusy(true); try { await runAutoStableTick(); await doRefresh(); } finally { setHealthBusy(false); }}}
                onRefresh={async () => { setHealthBusy(true); try { await doRefresh(); } finally { setHealthBusy(false); }}}
              />
            )}
            {activeView === "matrix" && <MatrixView groups={groups} busy={busy} onSelect={(g, p) => run(() => selectProxy(g, p), doRefresh)} />}
            {activeView === "rules" && <RuleEditorView groups={groups} busy={busy} />}
            {activeView === "policy" && <PolicyView groups={groups} busy={busy} />}
            {activeView === "logs" && <LogsView logs={logs} onClear={() => setLogs([])} />}
            {activeView === "settings" && (
              <SettingsView status={status} subscriptionUrl={subscriptionUrl} busy={busy}
                onSubscriptionUrlChange={setSubscriptionUrl}
                onSubmitSubscription={() => run(() => loadSubscription(subscriptionUrl), doRefresh)}
                onToggleSystemProxy={() => run(() => setSystemProxy(!status.systemProxyEnabled), doRefresh)}
                onToggleAutoStable={() => run(() => setAutoStableEnabled(!status.autoStableEnabled), doRefresh)}
              />
            )}
          </div>
          <ContextPanel status={status} autoStable={autoStable} groups={groups} busy={busy}
            onStart={() => run(startCore, doRefresh)}
            onStop={() => run(stopCore, doRefresh)}
            onRestart={() => run(restartCore, doRefresh)}
            onToggleSystemProxy={() => run(() => setSystemProxy(!status.systemProxyEnabled), doRefresh)}
            onToggleAutoStable={() => run(() => setAutoStableEnabled(!status.autoStableEnabled), doRefresh)}
          />
        </div>
      </main>
    </div>
  );
}
