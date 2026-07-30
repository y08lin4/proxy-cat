interface EmptyStateProps {
  message: string;
}

export function EmptyState({ message }: EmptyStateProps) {
  return (
    <div className="bg-[#fff4ef] rounded-2xl p-10 text-center text-brand-700">
      {message}
    </div>
  );
}
