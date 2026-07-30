interface StatusLineProps {
  label: string;
  value: string;
  tone?: "good" | "warn" | "muted";
}

const dotStyles: Record<string, string> = {
  good: "bg-green-500",
  warn: "bg-amber-500",
  muted: "bg-brand-300",
};

export function StatusLine({ label, value, tone = "muted" }: StatusLineProps) {
  return (
    <div className="flex justify-between items-center py-1">
      <span className="text-sm text-brand-700">{label}</span>
      <span className="flex items-center gap-1.5 text-sm font-medium">
        <span className={`w-1.5 h-1.5 rounded-full ${dotStyles[tone]}`} />
        {value}
      </span>
    </div>
  );
}
