import { useState } from "react";
import {
  Play,
  Square,
  Eye,
  HardDrive,
  Activity,
  TrendingUp,
  Clock,
  Tv,
  Minus,
  Plus,
  BarChart3,
  LineChart,
} from "lucide-react";
import { useMetrics, useEngineControl } from "../hooks/useIPC";
import { ViewersChart, BandwidthChart } from "../components/MetricsChart";
import MultiChannelCards from "../components/MultiChannelCards";
import ChannelSearch from "../components/ChannelSearch";

// Formats bytes into human-readable string.
function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1048576) return `${(bytes / 1024).toFixed(1)} KB`;
  if (bytes < 1073741824) return `${(bytes / 1048576).toFixed(1)} MB`;
  return `${(bytes / 1073741824).toFixed(2)} GB`;
}

// Formats nanosecond duration to human-readable string.
function formatUptime(ns: number): string {
  const totalSeconds = Math.floor(ns / 1e9);
  const hours = Math.floor(totalSeconds / 3600);
  const minutes = Math.floor((totalSeconds % 3600) / 60);
  const seconds = totalSeconds % 60;

  if (hours > 0) return `${hours}h ${minutes}m`;
  if (minutes > 0) return `${minutes}m ${seconds}s`;
  return `${seconds}s`;
}

export default function Dashboard() {
  const metrics = useMetrics();
  const { start, stop, setWorkers, loading } = useEngineControl();
  const [channel, setChannel] = useState("");
  const [workerCount, setWorkerCount] = useState(50);

  const isRunning = metrics?.engineState === "running";
  const isStopped = !metrics || metrics.engineState === "stopped";

  const handleStart = () => {
    if (channel.trim()) {
      start(channel.trim(), workerCount);
    }
  };

  return (
    <div className="h-full overflow-y-auto p-6 space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-100">Dashboard</h1>
          <p className="text-sm text-gray-500 mt-1">
            {metrics?.channel
              ? `Targeting: ${metrics.channel}`
              : "No active session"}
          </p>
        </div>

        {/* Engine state badge */}
        <div
          className={`
            badge text-sm px-3 py-1
            ${isRunning ? "badge-success" : isStopped ? "bg-gray-800 text-gray-400" : "badge-warning"}
          `}
        >
          {metrics?.engineState || "stopped"}
        </div>
      </div>

      {/* Quick Controls */}
      <div className="card">
        <div className="flex items-center gap-4">
          <ChannelSearch
            value={channel}
            onChange={setChannel}
            onSelect={(ch) => setChannel(ch.login)}
            disabled={isRunning}
            placeholder="Search channel or type name..."
          />

          <div className="flex items-center gap-2 bg-gray-800 rounded-lg p-1">
            <button
              className="p-1.5 hover:bg-gray-700 rounded transition-colors"
              onClick={() => setWorkerCount(Math.max(1, workerCount - 5))}
            >
              <Minus size={14} />
            </button>
            <span className="w-10 text-center text-sm font-mono">
              {workerCount}
            </span>
            <button
              className="p-1.5 hover:bg-gray-700 rounded transition-colors"
              onClick={() => setWorkerCount(Math.min(500, workerCount + 5))}
            >
              <Plus size={14} />
            </button>
          </div>

          {isStopped ? (
            <button
              className="btn-primary flex items-center gap-2"
              onClick={handleStart}
              disabled={!channel.trim() || loading}
            >
              <Play size={16} />
              Start
            </button>
          ) : (
            <button
              className="btn-danger flex items-center gap-2"
              onClick={stop}
              disabled={loading}
            >
              <Square size={16} />
              Stop
            </button>
          )}
        </div>

        {/* Worker count slider (when running) */}
        {isRunning && (
          <div className="mt-4 flex items-center gap-4">
            <span className="text-sm text-gray-400">Workers:</span>
            <input
              type="range"
              min={1}
              max={500}
              value={workerCount}
              onChange={(e) => {
                const val = parseInt(e.target.value);
                setWorkerCount(val);
                setWorkers(val);
              }}
              className="flex-1 accent-blue-500"
            />
            <span className="text-sm font-mono text-gray-300 w-8">
              {workerCount}
            </span>
          </div>
        )}
      </div>

      {/* Metrics Grid */}
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
        <MetricCard
          icon={<Eye className="text-blue-400" size={20} />}
          label="Active Viewers"
          value={metrics?.activeViewers ?? 0}
          subtitle={`/ ${metrics?.totalWorkers ?? 0} workers`}
          highlight
        />
        <MetricCard
          icon={<HardDrive className="text-emerald-400" size={20} />}
          label="Data Received"
          value={formatBytes(metrics?.bytesReceived ?? 0)}
        />
        <MetricCard
          icon={<Activity className="text-amber-400" size={20} />}
          label="Segments Fetched"
          value={(metrics?.segmentsFetched ?? 0).toLocaleString()}
        />
        <MetricCard
          icon={<TrendingUp className="text-cyan-400" size={20} />}
          label="Heartbeats"
          value={(metrics?.heartbeatsSent ?? 0).toLocaleString()}
        />
        <MetricCard
          icon={<Tv className="text-pink-400" size={20} />}
          label="Ads Watched"
          value={metrics?.adsWatched ?? 0}
        />
        <MetricCard
          icon={<Clock className="text-indigo-400" size={20} />}
          label="Uptime"
          value={formatUptime(metrics?.uptime ?? 0)}
        />
      </div>

      {/* Real-time Charts */}
      {isRunning && metrics && (
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
          <div className="card">
            <div className="flex items-center gap-2 mb-3">
              <LineChart size={16} className="text-blue-400" />
              <h3 className="text-sm font-semibold text-gray-400">
                Viewers Over Time
              </h3>
            </div>
            <div className="h-48">
              <ViewersChart metrics={metrics} />
            </div>
          </div>

          <div className="card">
            <div className="flex items-center gap-2 mb-3">
              <BarChart3 size={16} className="text-emerald-400" />
              <h3 className="text-sm font-semibold text-gray-400">
                Bandwidth (KB/s)
              </h3>
            </div>
            <div className="h-48">
              <BandwidthChart metrics={metrics} />
            </div>
          </div>
        </div>
      )}

      {/* Viewer Status Visualization */}
      {isRunning && metrics && (
        <div className="card">
          <h3 className="text-sm font-semibold text-gray-400 mb-3">
            Worker Status
          </h3>
          <div className="flex flex-wrap gap-1">
            {Array.from({ length: metrics.totalWorkers }).map((_, i) => (
              <div
                key={i}
                className={`
                  w-2.5 h-2.5 rounded-sm transition-colors
                  ${i < metrics.activeViewers ? "bg-emerald-500" : "bg-gray-700"}
                `}
                title={`Worker ${i + 1}: ${i < metrics.activeViewers ? "active" : "idle"}`}
              />
            ))}
          </div>
          <div className="flex items-center gap-4 mt-3 text-xs text-gray-500">
            <div className="flex items-center gap-1.5">
              <div className="w-2.5 h-2.5 rounded-sm bg-emerald-500" />
              Active ({metrics.activeViewers})
            </div>
            <div className="flex items-center gap-1.5">
              <div className="w-2.5 h-2.5 rounded-sm bg-gray-700" />
              Idle ({metrics.totalWorkers - metrics.activeViewers})
            </div>
          </div>
        </div>
      )}

      {/* Multi-Channel Cards (shows when multiple channels are running) */}
      <MultiChannelCards />
    </div>
  );
}

interface MetricCardProps {
  icon: React.ReactNode;
  label: string;
  value: string | number;
  subtitle?: string;
  highlight?: boolean;
}

function MetricCard({ icon, label, value, subtitle, highlight }: MetricCardProps) {
  return (
    <div className="card flex items-start gap-3">
      <div className="p-2 bg-gray-800 rounded-lg">{icon}</div>
      <div>
        <p className="text-xs text-gray-500 uppercase tracking-wider">
          {label}
        </p>
        <p
          className={`text-xl font-bold ${highlight ? "text-blue-400" : "text-gray-100"} mt-0.5`}
        >
          {value}
        </p>
        {subtitle && (
          <p className="text-xs text-gray-500 mt-0.5">{subtitle}</p>
        )}
      </div>
    </div>
  );
}
