// Custom hooks for IPC communication and real-time data.
import { useEffect, useState, useCallback, useRef } from "react";
import { ipc } from "../services/ipc";
import type { EngineMetrics, LogEntry } from "../types";

// useConnection manages the WebSocket connection lifecycle.
export function useConnection() {
  const [connected, setConnected] = useState(false);

  useEffect(() => {
    ipc
      .connect()
      .then(() => setConnected(true))
      .catch(() => setConnected(false));

    return () => ipc.disconnect();
  }, []);

  return connected;
}

// useMetrics subscribes to real-time engine metrics.
export function useMetrics() {
  const [metrics, setMetrics] = useState<EngineMetrics | null>(null);

  useEffect(() => {
    const unsub = ipc.on("event.metrics", (data) => {
      setMetrics(data as EngineMetrics);
    });
    return unsub;
  }, []);

  // Also poll on initial load
  useEffect(() => {
    ipc.call<EngineMetrics>("engine.status").then(setMetrics).catch(() => {});
  }, []);

  return metrics;
}

// useLogs subscribes to real-time log entries.
export function useLogs(maxEntries: number = 500) {
  const [logs, setLogs] = useState<LogEntry[]>([]);
  const logsRef = useRef<LogEntry[]>([]);

  useEffect(() => {
    // Load history first
    ipc
      .call<LogEntry[]>("logs.history")
      .then((history) => {
        logsRef.current = history || [];
        setLogs([...logsRef.current]);
      })
      .catch(() => {});

    // Subscribe to new entries
    const unsub = ipc.on("event.log", (data) => {
      const entry = data as LogEntry;
      logsRef.current = [...logsRef.current.slice(-maxEntries + 1), entry];
      setLogs([...logsRef.current]);
    });

    return unsub;
  }, [maxEntries]);

  return logs;
}

// useEngineControl provides engine start/stop actions.
export function useEngineControl() {
  const [loading, setLoading] = useState(false);

  const start = useCallback(async (channel: string, workers: number) => {
    setLoading(true);
    try {
      await ipc.call("engine.start", { channel, workers });
    } finally {
      setLoading(false);
    }
  }, []);

  const stop = useCallback(async () => {
    setLoading(true);
    try {
      await ipc.call("engine.stop");
    } finally {
      setLoading(false);
    }
  }, []);

  const setWorkers = useCallback(async (count: number) => {
    await ipc.call("engine.setWorkers", { count });
  }, []);

  return { start, stop, setWorkers, loading };
}
