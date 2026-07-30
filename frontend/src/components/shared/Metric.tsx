interface MetricProps {
  label: string;
  value: string;
}

export function Metric({ label, value }: MetricProps) {
  return (
    <div className="text-center px-3">
      <div className="text-2xl font-bold text-brand-900">{value}</div>
      <div className="text-xs text-brand-700 mt-1">{label}</div>
    </div>
  );
}
