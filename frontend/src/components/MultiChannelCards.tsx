import { useState, useEffect } from "react";
import {
  Play,
  Square,
  Eye,
  HardDrive,
  Activity,
  Tv,
  Minus,
  Plus,
  ChevronDown,
  ChevronUp,
} from "lucide-react";
import { ipc } from "../services/ipc";

interface ChannelStatus {
  channel: string;
  state: string;
  activeViewers: number;
  totalWorkers: number;
  metrics: {
    segmentsFetched: number;
    bytesReceived: number;
    heartbeatsSent: number;
    adsWatched: number;
    uptime: number;
  };
}

interface MultiStatus {
  channels: ChannelStatus[];
  count: number;
  aggregated: {
    activeViewers: number;
    totalWorkers: number;
    segmentsFetched: number;
    bytesReceived: number;
  };
}

function formatBytes(b: number): string {
  if (b < 1048576) return `${(b / 1024).toFixed(0)} KB`;
  if (b < 1073741824) return `${(b / 1048576).toFixed(1)} MB`;
  return `${(b / 1073741824).toFixed(2)} GB`;
}

export default function MultiChannelCards() {
  const [status, setStatus] = useState<MultiStatus | null>(null);
  const [expanded, setExpanded] = useState<string | null>(null);
  const [newChannel, setNewChannel] = useState("");
  const [newWorkers, setNewWorkers] = useState(25);

  // Poll multi-engine status
  useEffect(() => {
    const poll = () => {
      ipc.call<MultiStatus>("multi.status").then(setStatus).catch(() => {});
    };
    poll();
    const interval = setInterval(poll, 5000);
    return () => clearInterval(interval);
  }, []);

  const handleStart = async () => {
    if (!newChannel.trim()) return;
    try {
      await ipc.call("multi.start", { channel: newChannel, workers: newWorkers });
      setNewChannel("");
    } catch (err) {
      alert(`Failed: ${err}`);
    }
  };

  const handleStop = async (channel: string) => {
    await ipc.call("multi.stop", { channel });
  };

  if (!status || status.count === 0) {
    return null; // Don't render if no multi-channel activity
  }

  return (
    <div className="space-y-4">
      {/* Aggregated stats bar */}
      <div className="card flex items-center justify-between py-2 px-4">
        <div className="flex items-center gap-4 text-sm">
          <span className="text-gray-400">Multi-Channel</span>
          <span className="font-mono text-blue-400">
            {status.aggregated.activeViewers} viewers
          </span>
          <span className="font-mono text-emerald-400">
            {formatBytes(status.aggregated.bytesReceived)}
          </span>
        </div>
        <span className="badge-info">{status.count} channels</span>
      </div>

      {/* Channel cards */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-3">
        {status.channels.map((ch) => (
          <ChannelCard
            key={ch.channel}
            channel={ch}
            expanded={expanded === ch.channel}
            onToggle={() =>
              setExpanded(expanded === ch.channel ? null : ch.channel)
            }
            onStop={() => handleStop(ch.channel)}
          />
        ))}

        {/* Add channel card */}
        <div className="card border-dashed border-gray-700 flex flex-col items-center justify-center py-6 gap-3">
          <div className="flex items-center gap-2 w-full px-2">
            <input
              type="text"
              className="input-field flex-1 text-sm"
              placeholder="Add channel..."
              value={newChannel}
              onChange={(e) => setNewChannel(e.target.value)}
              onKeyDown={(e) => e.key === "Enter" && handleStart()}
            />
            <div className="flex items-center gap-1 bg-gray-800 rounded-lg p-0.5">
              <button
                className="p-1 hover:bg-gray-700 rounded"
                onClick={() => setNewWorkers(Math.max(1, newWorkers - 5))}
              >
                <Minus size={12} />
              </button>
              <span className="w-8 text-center text-xs font-mono">
                {newWorkers}
              </span>
              <button
                className="p-1 hover:bg-gray-700 rounded"
                onClick={() => setNewWorkers(newWorkers + 5)}
              >
                <Plus size={12} />
              </button>
            </div>
            <button
              className="btn-primary text-xs py-1.5 px-3"
              onClick={handleStart}
              disabled={!newChannel.trim()}
            >
              <Play size={12} />
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}

interface ChannelCardProps {
  channel: ChannelStatus;
  expanded: boolean;
  onToggle: () => void;
  onStop: () => void;
}

function ChannelCard({ channel, expanded, onToggle, onStop }: ChannelCardProps) {
  const isRunning = channel.state === "running";

  return (
    <div
      className={`card py-3 transition-all ${
        isRunning ? "border-emerald-500/20" : "border-gray-800 opacity-60"
      }`}
    >
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Tv size={16} className={isRunning ? "text-emerald-400" : "text-gray-500"} />
          <span className="font-semibold text-gray-200 text-sm">
            #{channel.channel}
          </span>
          <span className={`text-[10px] px-1.5 py-0.5 rounded ${
            isRunning ? "bg-emerald-500/20 text-emerald-400" : "bg-gray-800 text-gray-500"
          }`}>
            {channel.state}
          </span>
        </div>
        <div className="flex items-center gap-2">
          <span className="text-xs font-mono text-blue-400">
            {channel.activeViewers}/{channel.totalWorkers}
          </span>
          <button
            className="p-1 hover:bg-gray-800 rounded transition-colors text-gray-500 hover:text-red-400"
            onClick={onStop}
          >
            <Square size={12} />
          </button>
          <button
            className="p-1 hover:bg-gray-800 rounded transition-colors text-gray-500"
            onClick={onToggle}
          >
            {expanded ? <ChevronUp size={12} /> : <ChevronDown size={12} />}
          </button>
        </div>
      </div>

      {/* Expanded metrics */}
      {expanded && (
        <div className="grid grid-cols-4 gap-3 mt-3 pt-3 border-t border-gray-800/50 animate-fade-in">
          <MiniStat
            icon={<Eye size={12} className="text-blue-400" />}
            value={channel.activeViewers}
            label="Viewers"
          />
          <MiniStat
            icon={<Activity size={12} className="text-amber-400" />}
            value={channel.metrics.segmentsFetched}
            label="Segments"
          />
          <MiniStat
            icon={<HardDrive size={12} className="text-emerald-400" />}
            value={formatBytes(channel.metrics.bytesReceived)}
            label="Data"
          />
          <MiniStat
            icon={<Tv size={12} className="text-pink-400" />}
            value={channel.metrics.adsWatched}
            label="Ads"
          />
        </div>
      )}
    </div>
  );
}

function MiniStat({
  icon,
  value,
  label,
}: {
  icon: React.ReactNode;
  value: string | number;
  label: string;
}) {
  return (
    <div className="flex items-center gap-1.5">
      {icon}
      <div>
        <p className="text-xs font-semibold text-gray-300">{value}</p>
        <p className="text-[9px] text-gray-600">{label}</p>
      </div>
    </div>
  );
}
