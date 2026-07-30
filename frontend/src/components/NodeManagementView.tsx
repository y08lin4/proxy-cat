import { useState, useMemo } from "react";
import type { ProxyGroupView, ProxyView } from "../types";
import { PanelTitle, EmptyState, StatusTile, StatusLine } from "./shared";
import { proxyMeta, translateGroupType } from "../utils/translations";

interface NodeManagementViewProps {
  groups: ProxyGroupView[];
  busy: boolean;
}

export function NodeManagementView({ groups, busy }: NodeManagementViewProps) {
  const [search, setSearch] = useState("");
  const [pool, setPool] = useState<"all" | "front" | "exit">("all");

  // Extract all unique nodes from groups
  const allNodes = useMemo(() => {
    const seen = new Map<string, ProxyView>();
    groups.forEach(g => g.proxies.forEach(p => {
      if (!seen.has(p.name)) seen.set(p.name, p);
    }));
    return Array.from(seen.values()).sort((a, b) => a.name.localeCompare(b.name));
  }, [groups]);

  // Classify nodes: "front" = contains "relay"/"front"/"入口"/"中转", "exit" = everything else
  const nodePools = useMemo(() => {
    const fronts: ProxyView[] = [];
    const exits: ProxyView[] = [];
    allNodes.forEach(n => {
      if (/relay|front|入口|中转/i.test(n.name)) {
        fronts.push(n);
      } else {
        exits.push(n);
      }
    });
    return { fronts, exits, all: allNodes };
  }, [allNodes]);

  const filteredNodes = useMemo(() => {
    const source = pool === "front" ? nodePools.fronts : pool === "exit" ? nodePools.exits : nodePools.all;
    if (!search) return source;
    const q = search.toLowerCase();
    return source.filter(n => n.name.toLowerCase().includes(q) || (n.type || "").toLowerCase().includes(q));
  }, [nodePools, pool, search]);

  const healthyCount = useMemo(() => allNodes.filter(n => n.alive).length, [allNodes]);
  const avgLatency = useMemo(() => {
    const alive = allNodes.filter(n => n.alive && n.latencyMs && n.latencyMs > 0);
    return alive.length > 0 ? Math.round(alive.reduce((s, n) => s + (n.latencyMs ?? 0), 0) / alive.length) : 0;
  }, [allNodes]);

  return (
    <div className="flex flex-col gap-4">
      <PanelTitle title="节点管理" meta={`${allNodes.length} 节点 · ${healthyCount} 健康`} />

      {/* Summary tiles */}
      <div className="grid grid-cols-4 gap-3 max-[1080px]:grid-cols-2">
        <StatusTile label="总节点" value={String(allNodes.length)} tone="muted" />
        <StatusTile label="健康" value={`${healthyCount}/${allNodes.length}`} tone={healthyCount === allNodes.length ? "good" : "warn"} />
        <StatusTile label="平均延迟" value={`${avgLatency}ms`} tone="muted" />
        <StatusTile label="前用池" value={`${nodePools.fronts.length}`} tone="muted" />
      </div>

      {/* Toolbar */}
      <div className="flex gap-2">
        <input
          className="flex-1 rounded-2xl border-0 bg-[#fffaf7] px-4 py-2.5 text-sm shadow-[inset_0_0_0_1px_rgb(112_76_65/0.1)]"
          placeholder="搜索节点..."
          value={search}
          onChange={e => setSearch(e.target.value)}
        />
        <select value={pool} onChange={e => setPool(e.target.value as typeof pool)}
          className="rounded-2xl border-0 bg-[#fffaf7] px-3 py-2.5 text-sm shadow-[inset_0_0_0_1px_rgb(112_76_65/0.1)]">
          <option value="all">全部节点</option>
          <option value="front">前用节点池</option>
          <option value="exit">后用节点池</option>
        </select>
      </div>

      {/* Node grid */}
      {filteredNodes.length === 0 ? (
        <EmptyState message="没有匹配的节点。先在设置中加载订阅。" />
      ) : (
        <div className="grid grid-cols-[repeat(auto-fill,minmax(200px,1fr))] gap-3">
          {filteredNodes.map(node => (
            <div key={node.name}
              className={`rounded-2xl p-4 border transition-colors ${
                node.alive
                  ? "bg-[#fffaf7] border-[rgb(112_76_65/0.08)] shadow-sm"
                  : "bg-[#ffe7df]/30 border-[#8f321f]/20"
              }`}>
              <div className="flex items-center justify-between mb-2">
                <div className="font-bold text-sm text-brand-900 truncate">{node.name}</div>
                <span className={`w-2 h-2 rounded-full ${node.alive ? "bg-green-500" : "bg-red-400"}`} />
              </div>
              <div className="text-xs text-brand-500">
                <div>{node.type || "未知协议"}</div>
                {node.latencyMs !== undefined && node.latencyMs > 0 && (
                  <div className="font-bold text-brand-700 mt-0.5">{node.latencyMs}ms</div>
                )}
              </div>
              <div className="flex gap-1 mt-2">
                <span className={`inline-block px-1.5 py-0.5 rounded-full text-[10px] font-bold ${
                  /relay|front|入口|中转/i.test(node.name)
                    ? "bg-blue-50 text-blue-700"
                    : "bg-[#fff4ef] text-brand-600"
                }`}>
                  {/relay|front|入口|中转/i.test(node.name) ? "前用" : "后用"}
                </span>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
