import { Radio, Wifi, WifiOff, Activity, Clock, HardDrive, Zap } from "lucide-react";
import type { EngineMetrics } from "../types";

interface StatusBarProps {
  connected: boolean;
  metrics: EngineMetrics | null;
}

function formatBytes(b: number): string {
  if (b < 1024) return `${b} B`;
  if (b < 1048576) return `${(b / 1024).toFixed(0)} KB`;
  if (b < 1073741824) return `${(b / 1048576).toFixed(1)} MB`;
  return `${(b / 1073741824).toFixed(2)} GB`;
}

function formatUptime(ns: number): string {
  const s = Math.floor(ns / 1e9);
  const h = Math.floor(s / 3600);
  const m = Math.floor((s % 3600) / 60);
  if (h > 0) return `${h}h${m}m`;
  if (m > 0) return `${m}m`;
  return `${s}s`;
}

export default function StatusBar({ connected, metrics }: StatusBarProps) {
  const isRunning = metrics?.engineState === "running";

  return (
    <div className="h-7 bg-gray-900/80 border-t border-gray-800 flex items-center px-3 gap-4 text-[11px] text-gray-500 shrink-0 select-none">
      {/* Connection status */}
      <div className="flex items-center gap-1.5">
        {connected ? (
          <>
            <Wifi size={11} className="text-emerald-500" />
            <span className="text-emerald-500">Connected</span>
          </>
        ) : (
          <>
            <WifiOff size={11} className="text-red-400" />
            <span className="text-red-400">Disconnected</span>
          </>
        )}
      </div>

      <div className="w-px h-3 bg-gray-800" />

      {/* Engine state */}
      <div className="flex items-center gap-1.5">
        {isRunning ? (
          <Radio size={11} className="text-red-400 animate-pulse" />
        ) : (
          <Radio size={11} />
        )}
        <span className={isRunning ? "text-gray-300" : ""}>
          {metrics?.engineState || "stopped"}
        </span>
      </div>

      {isRunning && metrics && (
        <>
          <div className="w-px h-3 bg-gray-800" />

          {/* Viewers */}
          <div className="flex items-center gap-1.5">
            <Zap size={11} className="text-blue-400" />
            <span className="text-gray-300">
              {metrics.activeViewers}/{metrics.totalWorkers}
            </span>
          </div>

          <div className="w-px h-3 bg-gray-800" />

          {/* Bandwidth */}
          <div className="flex items-center gap-1.5">
            <HardDrive size={11} className="text-emerald-400" />
            <span>{formatBytes(metrics.bytesReceived)}</span>
          </div>

          <div className="w-px h-3 bg-gray-800" />

          {/* Segments */}
          <div className="flex items-center gap-1.5">
            <Activity size={11} className="text-amber-400" />
            <span>{metrics.segmentsFetched.toLocaleString()} seg</span>
          </div>

          <div className="w-px h-3 bg-gray-800" />

          {/* Uptime */}
          <div className="flex items-center gap-1.5">
            <Clock size={11} className="text-indigo-400" />
            <span>{formatUptime(metrics.uptime)}</span>
          </div>
        </>
      )}

      {/* Spacer */}
      <div className="flex-1" />

      {/* Channel */}
      {metrics?.channel && (
        <span className="text-gray-400 font-mono">#{metrics.channel}</span>
      )}

      {/* Keyboard hint */}
      <span className="text-gray-700 ml-2">
        Ctrl+1-8 nav · Esc stop
      </span>
    </div>
  );
}
