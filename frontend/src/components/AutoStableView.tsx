import { useState, useMemo } from "react";
import type { AutoStableGroupHealth } from "../types";
import type { AutoStableStatus } from "../types";
import { PanelTitle, StatusTile, EmptyState } from "./shared";
import { formatScore, formatLatency, formatFailureRate } from "../utils/formatters";
import { flattenHealth, summarizeHealth, HealthRow } from "../utils/filters";

interface AutoStableViewProps {
  autoStable: AutoStableStatus;
  healthBusy: boolean;
  onTick: () => Promise<void>;
  onRefresh: () => Promise<void>;
}

export function AutoStableView({ autoStable, healthBusy, onTick, onRefresh }: AutoStableViewProps) {
  const [groupName, setGroupName] = useState("");
  const groups = autoStable.health ?? [];
  const rows = useMemo<HealthRow[]>(() => flattenHealth(groups), [groups]);
  const summary = useMemo(() => summarizeHealth(rows), [rows]);

  const activeRows = groupName ? rows.filter(r => r.groupName === groupName) : rows;

  if (!autoStable.available) {
    return <EmptyState message="加载订阅后会生成自动稳定分组" />;
  }

  return (
    <div>
      <PanelTitle title="自动选择" meta={autoStable.running ? "运行中" : "已暂停"} />
      {/* Toggle */}
      <div className="flex items-center justify-between rounded-2xl bg-[#fffaf7] p-5 mb-4 shadow-[0_12px_32px_rgb(124_76_62/0.06)]">
        <div>
          <div className="font-bold text-brand-900">自动稳定选择</div>
          <div className="text-xs text-brand-500 mt-0.5">基于延迟和失败率自动选择最优节点</div>
        </div>
        <div className="flex gap-2">
          <button onClick={onRefresh} disabled={healthBusy} className="rounded-xl bg-[#fff4ef] px-4 py-2 text-sm text-brand-700 hover:bg-[#ffe4dc] disabled:opacity-50">
            {healthBusy ? "检测中" : "刷新检测"}
          </button>
          <button onClick={onTick} disabled={healthBusy} className="rounded-xl bg-[#dc7f69] px-4 py-2 text-sm text-white hover:bg-[#cd705d] disabled:opacity-50">
            执行选优
          </button>
        </div>
      </div>

      {/* Summary tiles */}
      <div className="grid grid-cols-4 gap-3 mb-4 max-[1080px]:grid-cols-2">
        <StatusTile label="最佳节点" value={summary.bestNode} tone="good" />
        <StatusTile label="健康节点" value={`${summary.healthy}/${summary.total}`} tone="good" />
        <StatusTile label="平均延迟" value={`${summary.averageLatency} ms`} tone="muted" />
        <StatusTile label="失败次数" value={String(summary.failures)} tone={summary.failures > 0 ? "warn" : "muted"} />
      </div>

      {/* Group selector */}
      {groups.length > 0 && (
        <select className="rounded-2xl border-0 bg-[#fffaf7] px-4 py-2.5 text-sm mb-4 shadow-[inset_0_0_0_1px_rgb(112_76_65/0.1)]" value={groupName} onChange={e => setGroupName(e.target.value)}>
          <option value="">全部节点</option>
          {groups.map(g => <option key={g.name} value={g.name}>{g.name} ({g.type})</option>)}
        </select>
      )}

      {/* Health table */}
      {rows.length === 0
        ? <EmptyState message="暂无健康检测数据" />
        : <div className="rounded-3xl bg-[#fffaf7] shadow-[0_12px_32px_rgb(124_76_62/0.06)] overflow-hidden">
            {activeRows.length === 0
              ? <div className="p-6 text-center text-sm text-brand-500">暂无数据</div>
              : activeRows.map(row => (
                  <div key={`${row.groupName}-${row.name}`} className="grid grid-cols-[minmax(180px,1.2fr)_repeat(4,minmax(110px,1fr))] gap-3 px-5 py-3 border-b border-[rgb(112_76_65/0.06)] items-center text-sm max-[1080px]:grid-cols-[1fr_repeat(2,1fr)] max-[560px]:grid-cols-1">
                    <div>
                      <div className="font-bold text-brand-900">{row.name}</div>
                      <div className="text-xs text-brand-500">{row.type || row.groupName}</div>
                    </div>
                    <div className="text-brand-700">
                      <div className="text-xs text-brand-400">评分</div>
                      <div className="font-bold">{formatScore(row)}</div>
                    </div>
                    <div className="text-brand-700">
                      <div className="text-xs text-brand-400">延迟</div>
                      <div>{formatLatency(row)}</div>
                    </div>
                    <div className="text-brand-700">
                      <div className="text-xs text-brand-400">失败率</div>
                      <div>{formatFailureRate(row)}</div>
                    </div>
                    <div className="text-brand-700">
                      <div className="text-xs text-brand-400">状态</div>
                      <span className={`inline-block px-2 py-0.5 rounded-full text-xs font-bold ${row.alive ? "bg-[#dff1df] text-[#3d7145]" : "bg-[#ffe7df] text-[#8f321f]"}`}>
                        {row.alive ? "正常" : "不可用"}
                      </span>
                    </div>
                  </div>
                ))}
          </div>}
    </div>
  );
}
