import { useState, useMemo } from "react";
import type { ProxyGroupView } from "../types";
import { PanelTitle, EmptyState } from "./shared";
import { translateGroupType, proxyMeta } from "../utils/translations";
import { filterGroups } from "../utils/filters";

interface ProxyViewPanelProps {
  groups: ProxyGroupView[];
  busy: boolean;
  onSelect: (groupName: string, proxyName: string) => Promise<void>;
}

export function ProxyViewPanel({ groups, busy, onSelect }: ProxyViewPanelProps) {
  const [query, setQuery] = useState("");
  const [view, setView] = useState<"all" | "selected" | "auto">("all");

  const filteredGroups = useMemo(() => filterGroups(groups, query, view), [groups, query, view]);

  if (groups.length === 0) {
    return <EmptyState message="先在设置中加载订阅，随后这里会出现代理组和节点" />;
  }

  return (
    <div>
      <PanelTitle title="代理" meta={`${groups.length} 组`} />
      <div className="flex gap-2 mb-4">
        <input className="flex-1 rounded-2xl border-0 bg-[#fffaf7] px-4 py-2.5 text-sm shadow-[inset_0_0_0_1px_rgb(112_76_65/0.1)] focus:outline-none focus:shadow-[inset_0_0_0_2px_#dc7f69]" placeholder="搜索代理组或节点..." value={query} onChange={e => setQuery(e.target.value)} />
        <select className="rounded-2xl border-0 bg-[#fffaf7] px-3 py-2.5 text-sm shadow-[inset_0_0_0_1px_rgb(112_76_65/0.1)]" value={view} onChange={e => setView(e.target.value as typeof view)}>
          <option value="all">全部</option>
          <option value="selected">已选择</option>
          <option value="auto">自动选择</option>
        </select>
      </div>
      {filteredGroups.length === 0
        ? <EmptyState message="没有匹配当前筛选条件的代理组" />
        : <div className="grid grid-cols-[repeat(auto-fill,minmax(280px,1fr))] gap-4">
            {filteredGroups.map(group => (
              <div key={group.name} className="rounded-3xl bg-[#fffaf7] p-5 shadow-[0_12px_32px_rgb(124_76_62/0.06),inset_0_1px_0_rgb(255_255_255/0.72)]">
                <div className="flex items-center justify-between mb-3">
                  <div>
                    <h3 className="font-bold text-brand-900">{group.name}</h3>
                    <span className="text-xs text-brand-500">{translateGroupType(group.type)}</span>
                  </div>
                  {group.selected && <span className="px-2 py-0.5 rounded-full text-xs bg-[#ffe4dc] text-brand-600 font-bold">{group.selected}</span>}
                </div>
                <div className="grid grid-cols-[repeat(auto-fill,minmax(100px,1fr))] gap-1.5">
                  {group.proxies.map(proxy => (
                    <button key={proxy.name} disabled={busy}
                      className={`rounded-xl px-3 py-2 text-xs text-left transition-colors ${proxy.name === group.selected ? "bg-[#ffd4c8] text-brand-900 font-bold" : "bg-[#fff5f0] text-brand-700 hover:bg-[#ffe4dc]"}`}
                      onClick={() => onSelect(group.name, proxy.name)}>
                      <div className="truncate font-bold">{proxy.name}</div>
                      <div className="opacity-70">{proxyMeta(proxy)}</div>
                    </button>
                  ))}
                </div>
              </div>
            ))}
          </div>}
    </div>
  );
}
