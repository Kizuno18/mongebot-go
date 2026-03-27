import { useState, useEffect } from "react";
import { History, Clock, Eye, HardDrive, Tv, TrendingUp } from "lucide-react";
import { ipc } from "../services/ipc";

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
      <div>
        <h1 className="text-2xl font-bold text-gray-100">Session History</h1>
        <p className="text-sm text-gray-500 mt-1">
          Past bot sessions with performance metrics
        </p>
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
  const isActive = !session.endedAt;
  const duration = formatDuration(session.startedAt, session.endedAt);

  return (
    <div
      className={`
        card animate-fade-in
        ${isActive ? "border-blue-500/30 ring-1 ring-blue-500/10" : ""}
      `}
    >
      {/* Header row */}
      <div className="flex items-center justify-between mb-3">
        <div className="flex items-center gap-3">
          <Tv size={18} className={isActive ? "text-blue-400" : "text-gray-500"} />
          <div>
            <span className="font-semibold text-gray-100">
              #{session.channel}
            </span>
            <span className="ml-2 badge-info text-[10px]">
              {session.platform}
            </span>
          </div>
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
