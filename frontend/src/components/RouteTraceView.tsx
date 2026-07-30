import { useState } from "react";

interface ChainNode {
  name: string;
  type: string;
  server: string;
  port: number;
  latencyMs?: number;
  alive: boolean;
  dialerName?: string;
}

interface ChainTrace {
  targetNode: string;
  hops: ChainNode[];
  totalHops: number;
}

export function RouteTraceView() {
  const [nodeName, setNodeName] = useState("");
  const [trace, setTrace] = useState<ChainTrace | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchTrace = async () => {
    if (!nodeName) return;
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`/api/v1/route-trace?node=${encodeURIComponent(nodeName)}`);
      if (!res.ok) throw new Error((await res.json()).message || res.statusText);
      setTrace(await res.json());
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="rounded-3xl bg-[#fffaf7] p-6 shadow-[0_12px_32px_rgb(124_76_62/0.06)]">
      <h3 className="text-lg font-bold text-brand-900 mb-4">路由链路追溯</h3>
      <div className="flex gap-2 mb-4">
        <input
          className="flex-1 rounded-2xl border-0 bg-[#fff5f0] px-4 py-2.5 text-sm shadow-[inset_0_0_0_1px_rgb(112_76_65/0.1)] focus:outline-none focus:shadow-[inset_0_0_0_2px_#dc7f69]"
          placeholder="输入节点名称，如 us-via-front-01"
          value={nodeName}
          onChange={e => setNodeName(e.target.value)}
          onKeyDown={e => e.key === "Enter" && fetchTrace()}
        />
        <button
          onClick={fetchTrace}
          disabled={loading || !nodeName}
          className="rounded-xl bg-[#dc7f69] px-5 py-2.5 text-sm text-white hover:bg-[#cd705d] disabled:opacity-50"
        >
          {loading ? "追溯中" : "追溯"}
        </button>
      </div>

      {error && <div className="bg-[#ffe7df] text-[#8f321f] rounded-xl px-4 py-3 text-sm mb-3">{error}</div>}

      {trace && (
        <div>
          <div className="text-xs text-brand-500 mb-2">共 {trace.totalHops} 跳</div>
          <div className="flex items-center gap-0 overflow-x-auto py-2">
            {trace.hops.map((hop, i) => (
              <div key={i} className="flex items-center gap-0">
                <div className={`rounded-2xl px-4 py-3 min-w-[140px] ${
                  hop.alive ? "bg-[#eef8ec] border border-[#3d7145]/20" : "bg-[#ffe7df] border border-[#8f321f]/20"
                }`}>
                  <div className="font-bold text-sm text-brand-900 truncate">{hop.name}</div>
                  <div className="text-xs text-brand-500">{hop.type}://{hop.server}:{hop.port}</div>
                  {hop.latencyMs !== undefined && hop.latencyMs > 0 && (
                    <div className="text-xs font-bold text-[#3d7145] mt-1">{hop.latencyMs}ms</div>
                  )}
                </div>
                {i < trace.hops.length - 1 && (
                  <div className="flex items-center px-1">
                    <div className="w-6 h-0.5 bg-brand-300 rounded" />
                    <div className="w-0 h-0 border-t-4 border-t-transparent border-b-4 border-b-transparent border-l-4 border-l-brand-300" />
                  </div>
                )}
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
