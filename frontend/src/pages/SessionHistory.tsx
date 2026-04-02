import { useState, useEffect } from "react";
import {
  History,
  Clock,
  Eye,
  HardDrive,
  Tv,
  TrendingUp,
  Download,
  ChevronDown,
  ChevronUp,
  Activity,
} from "lucide-react";
import { ipc } from "../services/ipc";
import { save } from "@tauri-apps/plugin-dialog";
import { writeTextFile } from "@tauri-apps/plugin-fs";
import {
  AreaChart,
  Area,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
} from "recharts";

interface Session {
  id: number;
  profileId: string;
  channel: string;
  platform: string;
  startedAt: string;
  endedAt: string | null;
  maxViewers: number;
  totalSegments: number;
  totalBytes: number;
  totalAds: number;
  totalHeartbeats: number;
  endReason: string | null;
}

interface MetricsSnapshot {
  timestamp: string;
  activeViewers: number;
  totalWorkers: number;
  segments: number;
  bytesReceived: number;
  heartbeats: number;
  adsWatched: number;
}

function formatBytes(b: number): string {
  if (b < 1024) return `${b} B`;
  if (b < 1048576) return `${(b / 1024).toFixed(1)} KB`;
  if (b < 1073741824) return `${(b / 1048576).toFixed(1)} MB`;
  return `${(b / 1073741824).toFixed(2)} GB`;
}

function formatDuration(start: string, end: string | null): string {
  const s = new Date(start).getTime();
  const e = end ? new Date(end).getTime() : Date.now();
  const diff = Math.floor((e - s) / 1000);

  const h = Math.floor(diff / 3600);
  const m = Math.floor((diff % 3600) / 60);
  if (h > 0) return `${h}h ${m}m`;
  return `${m}m`;
}

function formatDate(dateStr: string): string {
  const d = new Date(dateStr);
  return d.toLocaleDateString("en-US", {
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}

export default function SessionHistory() {
  const [sessions, setSessions] = useState<Session[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    ipc
      .call<Session[]>("sessions.recent", { limit: 50 })
      .then((data) => setSessions(data || []))
      .catch(() => {})
      .finally(() => setLoading(false));
  }, []);

  return (
    <div className="h-full overflow-y-auto p-6 space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-100">Session History</h1>
          <p className="text-sm text-gray-500 mt-1">
            Past bot sessions with performance metrics
          </p>
        </div>
        {sessions.length > 0 && (
          <div className="flex items-center gap-2">
            <button
              className="btn-ghost flex items-center gap-2 text-sm"
              onClick={async () => {
                try {
                  const result = await ipc.call<{ data: string }>("sessions.export", { format: "csv", limit: 100 });
                  const path = await save({
                    defaultPath: `mongebot-sessions-${new Date().toISOString().slice(0, 10)}.csv`,
                    filters: [{ name: "CSV", extensions: ["csv"] }],
                  });
                  if (path) {
                    await writeTextFile(path, result.data);
                  }
                } catch { /* ignore */ }
              }}
            >
              <Download size={14} />
              Export CSV
            </button>
            <button
              className="btn-ghost flex items-center gap-2 text-sm"
              onClick={async () => {
                try {
                  const result = await ipc.call<{ data: string }>("sessions.export", { format: "json", limit: 100 });
                  const path = await save({
                    defaultPath: `mongebot-sessions-${new Date().toISOString().slice(0, 10)}.json`,
                    filters: [{ name: "JSON", extensions: ["json"] }],
                  });
                  if (path) {
                    await writeTextFile(path, result.data);
                  }
                } catch { /* ignore */ }
              }}
            >
              <Download size={14} />
              Export JSON
            </button>
          </div>
        )}
      </div>

      {/* Sessions */}
      {loading ? (
        <div className="flex items-center justify-center py-20 text-gray-600">
          Loading sessions...
        </div>
      ) : sessions.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-20 text-gray-600">
          <History size={48} className="mb-3 opacity-30" />
          <p className="text-lg">No sessions recorded yet</p>
          <p className="text-sm mt-1">Start the engine to create your first session</p>
        </div>
      ) : (
        <div className="space-y-3">
          {sessions.map((session) => (
            <SessionCard key={session.id} session={session} />
          ))}
        </div>
      )}
    </div>
  );
}

function SessionCard({ session }: { session: Session }) {
  const [expanded, setExpanded] = useState(false);
  const [timeline, setTimeline] = useState<MetricsSnapshot[]>([]);
  const [loadingTimeline, setLoadingTimeline] = useState(false);

  const isActive = !session.endedAt;
  const duration = formatDuration(session.startedAt, session.endedAt);

  useEffect(() => {
    if (expanded && timeline.length === 0) {
      setLoadingTimeline(true);
      ipc.call<MetricsSnapshot[]>("sessions.timeline", { sessionId: session.id })
        .then((data) => setTimeline(data || []))
        .catch(() => {})
        .finally(() => setLoadingTimeline(false));
    }
  }, [expanded, session.id, timeline.length]);

  return (
    <div
      className={`
        card animate-fade-in
        ${isActive ? "border-blue-500/30 ring-1 ring-blue-500/10" : ""}
      `}
    >
      {/* Header row */}
      <div className="flex items-center justify-between mb-3">
        <div
          className="flex items-center gap-3 cursor-pointer"
          onClick={() => setExpanded(!expanded)}
        >
          <Tv size={18} className={isActive ? "text-blue-400" : "text-gray-500"} />
          <div>
            <span className="font-semibold text-gray-100">
              #{session.channel}
            </span>
            <span className="ml-2 badge-info text-[10px]">
              {session.platform}
            </span>
          </div>
          {expanded ? (
            <ChevronUp size={14} className="text-gray-500" />
          ) : (
            <ChevronDown size={14} className="text-gray-500" />
          )}
        </div>

        <div className="flex items-center gap-2">
          {isActive ? (
            <span className="badge-success flex items-center gap-1">
              <div className="w-1.5 h-1.5 rounded-full bg-emerald-400 animate-pulse" />
              Active
            </span>
          ) : (
            <span className="text-xs text-gray-600">
              {session.endReason || "completed"}
            </span>
          )}
        </div>
      </div>

      {/* Metrics row */}
      <div className="grid grid-cols-5 gap-4">
        <MiniMetric
          icon={<Clock size={14} className="text-indigo-400" />}
          label="Duration"
          value={duration}
        />
        <MiniMetric
          icon={<Eye size={14} className="text-blue-400" />}
          label="Peak Viewers"
          value={session.maxViewers.toString()}
        />
        <MiniMetric
          icon={<HardDrive size={14} className="text-emerald-400" />}
          label="Data"
          value={formatBytes(session.totalBytes)}
        />
        <MiniMetric
          icon={<TrendingUp size={14} className="text-amber-400" />}
          label="Segments"
          value={session.totalSegments.toLocaleString()}
        />
        <MiniMetric
          icon={<Tv size={14} className="text-pink-400" />}
          label="Ads"
          value={session.totalAds.toString()}
        />
      </div>

      {/* Expanded Timeline */}
      {expanded && (
        <div className="mt-4 pt-4 border-t border-gray-800/50 animate-fade-in">
          <h4 className="text-xs font-semibold text-gray-400 mb-2 flex items-center gap-2">
            <Activity size={12} />
            Session Timeline
          </h4>
          {loadingTimeline ? (
            <div className="h-32 flex items-center justify-center text-gray-600 text-sm">
              Loading timeline...
            </div>
          ) : timeline.length < 2 ? (
            <div className="h-32 flex items-center justify-center text-gray-600 text-sm">
              No timeline data available
            </div>
          ) : (
            <div className="h-40">
              <ResponsiveContainer width="100%" height="100%">
                <AreaChart data={timeline} margin={{ top: 5, right: 5, left: -20, bottom: 0 }}>
                  <defs>
                    <linearGradient id="viewerGradient" x1="0" y1="0" x2="0" y2="1">
                      <stop offset="5%" stopColor="#3b82f6" stopOpacity={0.3} />
                      <stop offset="95%" stopColor="#3b82f6" stopOpacity={0} />
                    </linearGradient>
                  </defs>
                  <CartesianGrid strokeDasharray="3 3" stroke="#1f2937" />
                  <XAxis
                    dataKey="timestamp"
                    stroke="#4b5563"
                    tick={{ fontSize: 9 }}
                    tickFormatter={(v) => v?.slice(11, 16) || ""}
                  />
                  <YAxis stroke="#4b5563" tick={{ fontSize: 9 }} />
                  <Tooltip
                    contentStyle={{
                      backgroundColor: "#111827",
                      border: "1px solid #374151",
                      borderRadius: "8px",
                      fontSize: "11px",
                    }}
                    labelFormatter={(v) => `Time: ${v}`}
                    formatter={(value: number, name: string) => [
                      name === "bytesReceived" ? formatBytes(value) : value,
                      name === "activeViewers" ? "Viewers" : name,
                    ]}
                  />
                  <Area
                    type="monotone"
                    dataKey="activeViewers"
                    stroke="#3b82f6"
                    fill="url(#viewerGradient)"
                    strokeWidth={2}
                    dot={false}
                    name="activeViewers"
                  />
                </AreaChart>
              </ResponsiveContainer>
            </div>
          )}
        </div>
      )}

      {/* Footer */}
      <div className="mt-3 pt-2 border-t border-gray-800/50 text-xs text-gray-600">
        Started {formatDate(session.startedAt)}
        {session.endedAt && ` — Ended ${formatDate(session.endedAt)}`}
      </div>
    </div>
  );
}

function MiniMetric({
  icon,
  label,
  value,
}: {
  icon: React.ReactNode;
  label: string;
  value: string;
}) {
  return (
    <div className="flex items-center gap-2">
      {icon}
      <div>
        <p className="text-[10px] text-gray-600 uppercase">{label}</p>
        <p className="text-sm font-semibold text-gray-300">{value}</p>
      </div>
    </div>
  );
}
