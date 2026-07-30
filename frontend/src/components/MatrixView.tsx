import { useState, useMemo } from "react";
import type { ProxyGroupView } from "../types";
import { PanelTitle, EmptyState } from "./shared";
import { translateGroupType } from "../utils/translations";

interface MatrixViewProps {
  groups: ProxyGroupView[];
  busy: boolean;
  onSelect?: (groupName: string, proxyName: string) => Promise<void>;
}

export function MatrixView({ groups, busy, onSelect }: MatrixViewProps) {
  const [query, setQuery] = useState("");

  const filteredGroups = useMemo(() => {
    if (!query) return groups;
    const q = query.toLowerCase();
    return groups.filter(g =>
      g.name.toLowerCase().includes(q) ||
      g.proxies.some(p => p.name.toLowerCase().includes(q))
    );
  }, [groups, query]);

  if (groups.length === 0) {
    return <EmptyState message="先在设置中加载订阅，随后这里会出现代理矩阵" />;
  }

  return (
    <div>
      <PanelTitle title="矩阵" meta={`${groups.length} 组`} />
      <div className="mb-4">
        <input
          className="w-full rounded-2xl border-0 bg-[#fffaf7] px-4 py-2.5 text-sm shadow-[inset_0_0_0_1px_rgb(112_76_65/0.1)] focus:outline-none focus:shadow-[inset_0_0_0_2px_#dc7f69]"
          placeholder="搜索代理组或节点..."
          value={query}
          onChange={e => setQuery(e.target.value)}
        />
      </div>

      {filteredGroups.length === 0
        ? <EmptyState message="没有匹配的代理组" />
        : (
          <div className="overflow-x-auto rounded-3xl bg-[#fffaf7] shadow-[0_12px_32px_rgb(124_76_62/0.06)]">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-[rgb(112_76_65/0.08)]">
                  <th className="sticky left-0 bg-[#fffaf7] px-4 py-3 text-left text-xs font-bold text-brand-500 uppercase tracking-wide">
                    代理组
                  </th>
                  <th className="px-3 py-3 text-center text-xs font-bold text-brand-500 uppercase tracking-wide w-20">
                    类型
                  </th>
                  <th className="px-3 py-3 text-left text-xs font-bold text-brand-500 uppercase tracking-wide">
                    当前选择
                  </th>
                  <th className="px-3 py-3 text-left text-xs font-bold text-brand-500 uppercase tracking-wide">
                    可用节点
                  </th>
                </tr>
              </thead>
              <tbody>
                {filteredGroups.map(group => (
                  <tr key={group.name} className="border-b border-[rgb(112_76_65/0.04)] hover:bg-[#fff5f0] transition-colors">
                    <td className="sticky left-0 bg-[#fffaf7] px-4 py-3 font-bold text-brand-900 whitespace-nowrap">
                      {group.name}
                    </td>
                    <td className="px-3 py-3 text-center text-xs text-brand-500">
                      {translateGroupType(group.type)}
                    </td>
                    <td className="px-3 py-3">
                      {group.selected
                        ? <span className="inline-block px-2.5 py-1 rounded-xl bg-[#ffe4dc] text-brand-700 font-bold text-xs">{group.selected}</span>
                        : <span className="text-xs text-brand-400">--</span>}
                    </td>
                    <td className="px-3 py-3">
                      <div className="flex flex-wrap gap-1.5">
                        {group.proxies.map(proxy => (
                          <button
                            key={proxy.name}
                            disabled={busy || !onSelect}
                            title={`${proxy.name}${proxy.latencyMs ? ` · ${proxy.latencyMs}ms` : ""}`}
                            onClick={() => onSelect?.(group.name, proxy.name)}
                            className={`inline-flex items-center gap-1 rounded-xl px-2.5 py-1.5 text-xs font-bold transition-colors ${
                              proxy.name === group.selected
                                ? "bg-[#ffd4c8] text-brand-900 shadow-[0_1px_3px_rgb(124_76_62/0.12)]"
                                : "bg-[#fff5f0] text-brand-700 hover:bg-[#ffe4dc]"
                            }`}
                          >
                            <span className="truncate max-w-[140px]">{proxy.name}</span>
                            <span className={`inline-block w-2 h-2 rounded-full flex-shrink-0 ${proxy.alive ? "bg-[#3d7145]" : "bg-[#8f321f]"}`} />
                            {proxy.latencyMs !== undefined && proxy.latencyMs > 0 && (
                              <span className="text-[10px] opacity-60 flex-shrink-0">{proxy.latencyMs}ms</span>
                            )}
                          </button>
                        ))}
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
    </div>
  );
}
