interface PanelTitleProps {
  title: string;
  meta: string;
}

export function PanelTitle({ title, meta }: PanelTitleProps) {
  return (
    <div className="flex items-center justify-between mb-4">
      <h2 className="text-[21px] font-normal text-brand-900">{title}</h2>
      <span className="text-xs text-brand-700">{meta}</span>
    </div>
  );
}
