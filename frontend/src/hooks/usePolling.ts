import { useEffect, useRef } from "react";

export function usePolling(callback: () => Promise<void>, intervalMs: number) {
  const callbackRef = useRef(callback);
  callbackRef.current = callback;

  useEffect(() => {
    callbackRef.current(); // immediate first call
    const id = window.setInterval(() => callbackRef.current(), intervalMs);
    return () => window.clearInterval(id);
  }, [intervalMs]);
}
