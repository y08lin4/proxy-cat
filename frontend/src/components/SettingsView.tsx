import type { AppStatus } from "../types";
import { PanelTitle, StatusLine } from "./shared";

interface SettingsViewProps {
  status: AppStatus;
  subscriptionUrl: string;
  busy: boolean;
  onSubscriptionUrlChange: (url: string) => void;
  onSubmitSubscription: () => Promise<void>;
  onToggleSystemProxy: () => Promise<void>;
  onToggleAutoStable: () => Promise<void>;
}

export function SettingsView({ status, subscriptionUrl, busy, onSubscriptionUrlChange, onSubmitSubscription, onToggleSystemProxy, onToggleAutoStable }: SettingsViewProps) {
  return (
    <div>
      <PanelTitle title="设置" meta="" />
      <div className="grid grid-cols-[minmax(0,1.2fr)_minmax(320px,0.8fr)] gap-4 max-[1080px]:grid-cols-1">
        {/* Subscription form */}
        <div className="rounded-3xl bg-[#fffaf7] p-6 shadow-[0_12px_32px_rgb(124_76_62/0.06)]">
          <h3 className="font-bold text-brand-900 mb-3">订阅管理</h3>
          <div className="flex gap-2">
            <input className="flex-1 rounded-2xl border-0 bg-[#fff5f0] px-4 py-2.5 text-sm shadow-[inset_0_0_0_1px_rgb(112_76_65/0.1)] focus:outline-none focus:shadow-[inset_0_0_0_2px_#dc7f69]" placeholder="输入订阅地址 (Clash/Mihomo YAML)" value={subscriptionUrl} onChange={e => onSubscriptionUrlChange(e.target.value)}
              onKeyDown={e => e.key === "Enter" && onSubmitSubscription()} />
            <button onClick={onSubmitSubscription} disabled={busy || !subscriptionUrl} className="rounded-xl bg-[#dc7f69] px-5 py-2.5 text-sm text-white hover:bg-[#cd705d] disabled:opacity-50 whitespace-nowrap">
              {busy ? "加载中" : "加载订阅"}
            </button>
          </div>
          {status.activeProfileName && (
            <p className="text-xs text-brand-500 mt-3">当前配置: {status.activeProfileName}</p>
          )}
        </div>

        {/* Toggles */}
        <div className="rounded-3xl bg-[#fffaf7] p-6 shadow-[0_12px_32px_rgb(124_76_62/0.06)] space-y-3">
          <h3 className="font-bold text-brand-900 mb-3">快捷开关</h3>
          <div className="flex justify-between items-center py-2">
            <span className="text-sm text-brand-700">系统代理</span>
            <button onClick={onToggleSystemProxy} disabled={busy}
              className={`relative w-12 h-7 rounded-full transition-colors ${status.systemProxyEnabled ? "bg-[#dc7f69]" : "bg-brand-300"}`}>
              <span className={`absolute top-0.5 w-6 h-6 rounded-full bg-white shadow transition-transform ${status.systemProxyEnabled ? "translate-x-[22px]" : "translate-x-0.5"}`} />
            </button>
          </div>
          <div className="flex justify-between items-center py-2">
            <span className="text-sm text-brand-700">自动选择</span>
            <button onClick={onToggleAutoStable} disabled={busy}
              className={`relative w-12 h-7 rounded-full transition-colors ${status.autoStableEnabled ? "bg-[#dc7f69]" : "bg-brand-300"}`}>
              <span className={`absolute top-0.5 w-6 h-6 rounded-full bg-white shadow transition-transform ${status.autoStableEnabled ? "translate-x-[22px]" : "translate-x-0.5"}`} />
            </button>
          </div>
          <StatusLine label="控制器地址" value={status.controllerAddress} />
          <StatusLine label="内核状态" value={status.coreRunning ? "运行中" : "已停止"} tone={status.coreRunning ? "good" : "muted"} />
        </div>
      </div>
    </div>
  );
}
