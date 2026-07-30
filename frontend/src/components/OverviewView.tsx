import type { LogLine, ProxyGroupView, AutoStableGroupHealth } from "../types";
import type { AppStatus, ConnectionStatus } from "../types";
import { Metric, StatusTile, EmptyState } from "./shared";
import { formatBytes, formatRelativeTime } from "../utils/formatters";
import { HealthRow, flattenHealth, summarizeHealth } from "../utils/filters";

interface OverviewViewProps {
  status: AppStatus;
  connection: ConnectionStatus;
  groups: ProxyGroupView[];
  logs: LogLine[];
  health?: AutoStableGroupHealth[];
}

export function OverviewView({ status, connection, groups, logs }: OverviewViewProps) {
  const selectedNodes = groups.filter(g => g.selected).map(g => `${g.name}：${g.selected}`);
  const healthRows: HealthRow[] = [];
  const summary = summarizeHealth(healthRows);
  const recentLogs = logs.slice(-5).reverse();

  return (
    <div className="flex flex-col gap-4">
      {/* Hero */}
      <div className="grid grid-cols-[minmax(0,1.1fr)_minmax(280px,0.9fr)] gap-4 max-[1080px]:grid-cols-1">
        <div className="rounded-3xl bg-[#fffaf7] p-7 shadow-[0_18px_44px_rgb(124_76_62/0.08),inset_0_1px_0_rgb(255_255_255/0.72)]">
          <div className="flex items-center gap-2 mb-2">
            <span className={`inline-block px-2.5 py-0.5 rounded-full text-xs font-bold ${status.coreRunning ? "bg-[#dff1df] text-[#3d7145]" : "bg-[#ffe7df] text-[#8f321f]"}`}>
              {status.coreRunning ? "内核运行中" : "内核未启动"}
            </span>
          </div>
          <h3 className="text-[30px] font-normal text-brand-900 mb-1">{status.activeProfileName || "Proxy-Cat"}</h3>
          <p className="text-sm text-brand-700">{status.controllerAddress}</p>
        </div>
        <div className="rounded-3xl bg-[#fffaf7] p-7 shadow-[0_18px_44px_rgb(124_76_62/0.08),inset_0_1px_0_rgb(255_255_255/0.72)] flex items-center justify-around">
          <Metric label="连接数" value={String(connection.connectionCount)} />
          <Metric label="上传" value={formatBytes(connection.uploadTotal)} />
          <Metric label="下载" value={formatBytes(connection.downloadTotal)} />
        </div>
      </div>

      {/* Status tiles */}
      <div className="grid grid-cols-4 gap-3 max-[1080px]:grid-cols-2 max-[560px]:grid-cols-1">
        <StatusTile label="内核状态" value={status.coreRunning ? "运行中" : "已停止"} tone={status.coreRunning ? "good" : "muted"} />
        <StatusTile label="系统代理" value={status.systemProxyEnabled ? "已开启" : "已关闭"} tone={status.systemProxyEnabled ? "good" : "muted"} />
        <StatusTile label="自动选择" value={status.autoStableEnabled ? "已启用" : "未启用"} tone="muted" />
        <StatusTile label="代理组" value={`${groups.length} 组`} tone="muted" />
      </div>

      {/* Dashboard grid */}
      <div className="grid grid-cols-[1.2fr_0.9fr_1fr] gap-4 max-[1080px]:grid-cols-1">
        <div className="rounded-3xl bg-[#fffaf7] p-6 shadow-[0_18px_44px_rgb(124_76_62/0.08),inset_0_1px_0_rgb(255_255_255/0.72)]">
          <h3 className="text-lg font-normal text-brand-900 mb-3">当前选择</h3>
          {selectedNodes.length > 0
            ? <ul className="space-y-1.5 text-sm text-brand-800">{selectedNodes.slice(0, 4).map((n, i) => <li key={i}>{n}</li>)}</ul>
            : <EmptyState message="还没有已选择的代理组" />}
        </div>
        <div className="rounded-3xl bg-[#fffaf7] p-6 shadow-[0_18px_44px_rgb(124_76_62/0.08),inset_0_1px_0_rgb(255_255_255/0.72)]">
          <h3 className="text-lg font-normal text-brand-900 mb-3">稳定性摘要</h3>
          <div className="space-y-2 text-sm text-brand-700">
            <div className="flex justify-between"><span>健康节点</span><span className="font-bold text-[#3d7145]">{summary.healthy}/{summary.total}</span></div>
            <div className="flex justify-between"><span>平均延迟</span><span>{summary.averageLatency} ms</span></div>
          </div>
        </div>
        <div className="rounded-3xl bg-[#fffaf7] p-6 shadow-[0_18px_44px_rgb(124_76_62/0.08),inset_0_1px_0_rgb(255_255_255/0.72)]">
          <h3 className="text-lg font-normal text-brand-900 mb-3">最近日志</h3>
          {recentLogs.length > 0
            ? <div className="space-y-1 text-xs">{recentLogs.map((l, i) => <div key={i} className="truncate text-brand-700"><span className="text-brand-400">{formatRelativeTime(l.time)}</span> {l.message}</div>)}</div>
            : <EmptyState message="暂无日志" />}
        </div>
      </div>
    </div>
  );
}
