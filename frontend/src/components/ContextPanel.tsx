import type { AppStatus, AutoStableStatus } from "../types";
import type { ProxyGroupView } from "../types";
import { StatusLine } from "./shared";
import { summarizeHealth, flattenHealth } from "../utils/filters";

interface ContextPanelProps {
  status: AppStatus;
  autoStable: AutoStableStatus;
  groups: ProxyGroupView[];
  busy: boolean;
  onStart: () => Promise<void>;
  onStop: () => Promise<void>;
  onRestart: () => Promise<void>;
  onToggleSystemProxy: () => Promise<void>;
  onToggleAutoStable: () => Promise<void>;
  onRunTick?: () => Promise<void>;
}

export function ContextPanel({ status, autoStable, groups, busy, onStart, onStop, onRestart, onToggleSystemProxy, onToggleAutoStable }: ContextPanelProps) {
  const selectedNodes = groups.filter(g => g.selected).map(g => `${g.name}：${g.selected}`);
  const rows = flattenHealth(autoStable.health ?? []);
  const summary = summarizeHealth(rows);

  return (
    <aside aria-label="运行状态" className="sticky top-[22px] flex flex-col gap-4">
      {/* Run control */}
      <div className="rounded-3xl bg-[#fffaf7] p-5 shadow-[0_12px_32px_rgb(124_76_62/0.06)]">
        <h3 className="font-bold text-brand-900 mb-3">运行控制</h3>
        <div className="flex flex-wrap gap-2">
          <button onClick={onStart} disabled={busy || status.coreRunning} className="rounded-xl bg-[#dc7f69] px-4 py-2 text-sm text-white hover:bg-[#cd705d] disabled:opacity-50">
            启动内核
          </button>
          <button onClick={onStop} disabled={busy || !status.coreRunning} className="rounded-xl bg-[#fff4ef] px-4 py-2 text-sm text-brand-700 hover:bg-[#ffe4dc] disabled:opacity-50">
            停止
          </button>
          <button onClick={onRestart} disabled={busy} className="rounded-xl bg-[#fff4ef] px-4 py-2 text-sm text-brand-700 hover:bg-[#ffe4dc] disabled:opacity-50">
            重启
          </button>
        </div>
      </div>

      {/* Real-time status */}
      <div className="rounded-3xl bg-[#fffaf7] p-5 shadow-[0_12px_32px_rgb(124_76_62/0.06)]">
        <h3 className="font-bold text-brand-900 mb-3">实时状态</h3>
        <div className="space-y-2">
          <StatusLine label="系统代理" value={status.systemProxyEnabled ? "已开启" : "已关闭"} tone={status.systemProxyEnabled ? "good" : "muted"} />
          <StatusLine label="自动选择" value={autoStable.running ? "运行中" : "已暂停"} tone={autoStable.running ? "good" : "muted"} />
          <StatusLine label="代理组" value={`${groups.length} 组`} tone="muted" />
        </div>
      </div>

      {/* Quick toggles */}
      <div className="rounded-3xl bg-[#fffaf7] p-5 shadow-[0_12px_32px_rgb(124_76_62/0.06)]">
        <h3 className="font-bold text-brand-900 mb-3">快捷开关</h3>
        <div className="space-y-3">
          <div className="flex justify-between items-center">
            <span className="text-sm text-brand-700">系统代理</span>
            <button onClick={onToggleSystemProxy} disabled={busy}
              className={`relative w-12 h-7 rounded-full transition-colors ${status.systemProxyEnabled ? "bg-[#dc7f69]" : "bg-brand-300"}`}>
              <span className={`absolute top-0.5 w-6 h-6 rounded-full bg-white shadow transition-transform ${status.systemProxyEnabled ? "translate-x-[22px]" : "translate-x-0.5"}`} />
            </button>
          </div>
          <div className="flex justify-between items-center">
            <span className="text-sm text-brand-700">自动选择</span>
            <button onClick={onToggleAutoStable} disabled={busy}
              className={`relative w-12 h-7 rounded-full transition-colors ${status.autoStableEnabled ? "bg-[#dc7f69]" : "bg-brand-300"}`}>
              <span className={`absolute top-0.5 w-6 h-6 rounded-full bg-white shadow transition-transform ${status.autoStableEnabled ? "translate-x-[22px]" : "translate-x-0.5"}`} />
            </button>
          </div>
        </div>
      </div>

      {/* Selected exit nodes */}
      <div className="rounded-3xl bg-[#fffaf7] p-5 shadow-[0_12px_32px_rgb(124_76_62/0.06)]">
        <h3 className="font-bold text-brand-900 mb-3">当前出口</h3>
        {selectedNodes.length > 0
          ? <ul className="space-y-1.5 text-sm text-brand-800">{selectedNodes.slice(0, 4).map((n, i) => <li key={i}>{n}</li>)}</ul>
          : <div className="text-sm text-brand-400">暂无已选择分组</div>}
      </div>

      {/* Stability */}
      <div className="rounded-3xl bg-[#fffaf7] p-5 shadow-[0_12px_32px_rgb(124_76_62/0.06)]">
        <h3 className="font-bold text-brand-900 mb-3">稳定性</h3>
        <div className="space-y-2 text-sm">
          <StatusLine label="健康节点" value={`${summary.healthy}/${summary.total}`} tone={summary.healthy > 0 ? "good" : "muted"} />
          <StatusLine label="平均延迟" value={`${summary.averageLatency} ms`} tone="muted" />
        </div>
      </div>
    </aside>
  );
}
