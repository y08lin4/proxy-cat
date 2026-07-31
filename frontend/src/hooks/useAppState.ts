import { useCallback, useState } from "react";
import type { LogLine, ProxyGroupView } from "../types";
import type { AppStatus, AutoStableStatus, ConnectionStatus } from "../types";
import { getAutoStableStatus, getConnectionStatus, getLogs, getProxyGroups, getStatus } from "../api/client";

const emptyStatus: AppStatus = {
  coreRunning: false, systemProxyEnabled: false, autoStableEnabled: false,
  activeProfileName: "", controllerAddress: "127.0.0.1:9090",
};

const emptyConnection: ConnectionStatus = {
  coreRunning: false, uploadTotal: 0, downloadTotal: 0, connectionCount: 0,
};

const emptyAutoStable: AutoStableStatus = {
  enabled: false, available: false, running: false, health: [],
};

export function useAppState() {
  const [status, setStatus] = useState<AppStatus>(emptyStatus);
  const [autoStable, setAutoStable] = useState<AutoStableStatus>(emptyAutoStable);
  const [connection, setConnection] = useState<ConnectionStatus>(emptyConnection);
  const [groups, setGroups] = useState<ProxyGroupView[]>([]);
  const [logs, setLogs] = useState<LogLine[]>([]);

  const refresh = useCallback(async () => {
    try {
      const [s, g, l, c] = await Promise.all([
        getStatus(),
        getProxyGroups(),
        getLogs(80),
        getConnectionStatus(),
      ]);
      setStatus(s ?? emptyStatus);
      setGroups(g ?? []);
      setLogs(l ?? []);
      setConnection(c ?? emptyConnection);
    } catch (err) {
      throw err;
    }
  }, []);

  const refreshHealth = useCallback(async () => {
    try {
      const a = await getAutoStableStatus();
      setAutoStable(a ?? emptyAutoStable);
    } catch { /* health refresh is non-critical */ }
  }, []);

  return { status, setStatus, autoStable, setAutoStable, connection, groups, logs, setLogs, refresh, refreshHealth };
}
