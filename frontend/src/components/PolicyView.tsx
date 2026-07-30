import { useState, useMemo } from "react";
import type { ProxyGroupView } from "../types";
import { PanelTitle, EmptyState } from "./shared";
import { translateGroupType } from "../utils/translations";

interface PolicyViewProps {
  groups: ProxyGroupView[];
  busy: boolean;
}

function PolicyCard({ group }: { group: ProxyGroupView }) {
  return (
    <div className="rounded-2xl bg-[#fffaf7] p-4 shadow-[0_4px_16px_rgb(124_76_62/0.04)]">
      <div className="flex items-center justify-between mb-3">
        <div className="flex items-center gap-2">
          <span className="text-sm font-bold text-brand-900">{group.name}</span>
          <span className="text-xs text-brand-400">{translateGroupType(group.type)}</span>
        </div>
        <span className="text-xs text-brand-500">选中: {group.selected || "—"}</span>
      </div>

      <div className="grid grid-cols-2 gap-x-4 gap-y-1.5">
        {group.strategy !== undefined && (
          <>
            <span className="text-xs text-brand-400">策略</span>
            <span className="text-xs text-brand-700 font-medium">{group.strategy}</span>
          </>
        )}
        {group.interval !== undefined && (
          <>
            <span className="text-xs text-brand-400">间隔</span>
            <span className="text-xs text-brand-700 font-medium">{group.interval}s</span>
          </>
        )}
        {group.tolerance !== undefined && (
          <>
            <span className="text-xs text-brand-400">容差</span>
            <span className="text-xs text-brand-700 font-medium">{group.tolerance}ms</span>
          </>
        )}
        {group.lazy !== undefined && (
          <>
            <span className="text-xs text-brand-400">延迟解析</span>
            <span className="text-xs text-brand-700 font-medium">{group.lazy ? "是" : "否"}</span>
          </>
        )}
        {group.stickyMaxAge !== undefined && (
          <>
            <span className="text-xs text-brand-400">粘性最大时间</span>
            <span className="text-xs text-brand-700 font-medium">{group.stickyMaxAge}s</span>
          </>
        )}
        {group.testUrl !== undefined && (
          <>
            <span className="text-xs text-brand-400">测试 URL</span>
            <span className="text-xs text-brand-700 font-medium truncate">{group.testUrl}</span>
          </>
        )}
      </div>

      {group.chainNodes && group.chainNodes.length > 0 && (
        <div className="mt-3 pt-3 border-t border-[rgb(112_76_65/0.08)]">
          <span className="text-xs text-brand-400 block mb-1">链式节点</span>
          <div className="flex flex-wrap gap-1">
            {group.chainNodes.map((hop, i) => (
              <span key={i} className="inline-block px-2 py-0.5 rounded-full bg-[#fff4ef] text-xs text-brand-700">
                {hop.proxyName}
                {hop.dialerProxy && <span className="text-brand-400"> → {hop.dialerProxy}</span>}
              </span>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}

export function PolicyView({ groups, busy }: PolicyViewProps) {
  const [query, setQuery] = useState("");

  const policyGroups = useMemo(() => {
    const filtered = query
      ? groups.filter(g => {
        const q = query.toLowerCase();
        return (
          g.name.toLowerCase().includes(q) ||
          (g.type && g.type.toLowerCase().includes(q)) ||
          (g.strategy && g.strategy.toLowerCase().includes(q)) ||
          (g.testUrl && g.testUrl.toLowerCase().includes(q))
        );
      })
      : groups;
    return filtered;
  }, [groups, query]);

  if (groups.length === 0) {
    return <EmptyState message="先在设置中加载订阅，随后这里会出现策略组配置" />;
  }

  return (
    <div>
      <PanelTitle title="策略" meta={`${policyGroups.length} 组`} />

      <div className="mb-4">
        <input
          className="w-full rounded-2xl border-0 bg-[#fffaf7] px-4 py-2.5 text-sm shadow-[inset_0_0_0_1px_rgb(112_76_65/0.1)] focus:outline-none focus:shadow-[inset_0_0_0_2px_#dc7f69]"
          placeholder="搜索策略组..."
          value={query}
          onChange={e => setQuery(e.target.value)}
        />
      </div>

      {policyGroups.length === 0
        ? <EmptyState message="没有匹配的策略组" />
        : (
          <div className="space-y-3">
            {policyGroups.map(g => (
              <PolicyCard key={g.name} group={g} />
            ))}
          </div>
        )}
    </div>
  );
}
