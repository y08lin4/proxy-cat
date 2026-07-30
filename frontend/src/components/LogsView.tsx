import { useState, useMemo } from "react";
import type { LogLine } from "../types";
import { PanelTitle, EmptyState } from "./shared";
import { translateLogLevel, logTone } from "../utils/translations";
import { formatRelativeTime } from "../utils/formatters";
import { filterLogs, uniqueLogLevels } from "../utils/filters";

interface LogsViewProps {
  logs: LogLine[];
  onClear: () => void;
}

export function LogsView({ logs, onClear }: LogsViewProps) {
  const [level, setLevel] = useState("all");
  const [query, setQuery] = useState("");
  const levels = useMemo(() => uniqueLogLevels(logs), [logs]);
  const filteredLogs = useMemo(() => filterLogs(logs, level, query), [logs, level, query]);

  return (
    <div>
      <PanelTitle title="日志" meta={`${logs.length} 条`} />
      <div className="flex gap-2 mb-4">
        <select className="rounded-2xl border-0 bg-[#fffaf7] px-3 py-2.5 text-sm shadow-[inset_0_0_0_1px_rgb(112_76_65/0.1)]" value={level} onChange={e => setLevel(e.target.value)}>
          <option value="all">全部级别</option>
          {levels.map(l => <option key={l} value={l}>{translateLogLevel(l)}</option>)}
        </select>
        <input className="flex-1 rounded-2xl border-0 bg-[#fffaf7] px-4 py-2.5 text-sm shadow-[inset_0_0_0_1px_rgb(112_76_65/0.1)] focus:outline-none focus:shadow-[inset_0_0_0_2px_#dc7f69]" placeholder="搜索日志..." value={query} onChange={e => setQuery(e.target.value)} />
        <button onClick={onClear} className="rounded-xl bg-[#fff4ef] px-4 py-2.5 text-sm text-brand-700 hover:bg-[#ffe4dc]">清空</button>
      </div>
      {logs.length === 0
        ? <EmptyState message="暂无日志" />
        : filteredLogs.length === 0
          ? <EmptyState message="没有匹配的日志" />
          : <div className="rounded-3xl bg-[#fffaf7] shadow-[0_12px_32px_rgb(124_76_62/0.06)] overflow-hidden max-h-[600px] overflow-y-auto">
              {filteredLogs.map((l, i) => {
                const tone = logTone(l.level);
                const toneStyles = { bad: "bg-[#ffe7df] text-[#8f321f]", warn: "bg-[#fff0d9] text-amber-800", faded: "bg-[#fff5f0] text-brand-400", info: "bg-[#eef8ec] text-brand-700" };
                return (
                  <div key={i} className={`grid grid-cols-[88px_72px_minmax(0,1fr)] gap-3 px-5 py-2.5 border-b border-[rgb(112_76_65/0.06)] items-center text-xs max-[560px]:grid-cols-1`}>
                    <span className="text-brand-400">{formatRelativeTime(l.time)}</span>
                    <span className={`inline-block px-2 py-0.5 rounded-full text-xs font-extrabold text-center ${toneStyles[tone]}`}>{translateLogLevel(l.level)}</span>
                    <span className="text-brand-700 overflow-wrap-anywhere break-all">{l.message}</span>
                  </div>
                );
              })}
            </div>}
    </div>
  );
}
