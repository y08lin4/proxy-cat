import { useCallback, useState } from "react";

export function toMessage(err: unknown): string {
  if (err instanceof Error) return err.message;
  return String(err ?? "");
}

export function useActionRunner() {
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const run = useCallback(async (action: () => Promise<void>, onSuccess?: () => Promise<void>) => {
    setBusy(true);
    setError(null);
    try {
      await action();
      if (onSuccess) await onSuccess();
    } catch (err) {
      setError(toMessage(err));
    } finally {
      setBusy(false);
    }
  }, []);

  return { busy, error, run, clearError: () => setError(null) };
}
