interface StatusTileProps {
  label: string;
  value: string | number;
  tone?: "good" | "warn" | "muted";
}

const toneStyles: Record<string, string> = {
  good: "bg-[#eef8ec] text-[#3d7145]",
  warn: "bg-[#fff0d9] text-amber-800",
  muted: "bg-[#fff4ef] text-brand-700",
};

export function StatusTile({ label, value, tone = "muted" }: StatusTileProps) {
  return (
    <div className={`rounded-2xl p-4 ${toneStyles[tone]}`}>
      <div className="text-xs opacity-70 mb-1">{label}</div>
      <div className="text-xl font-bold">{value}</div>
    </div>
  );
}
