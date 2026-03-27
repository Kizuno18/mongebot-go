import { useState, useEffect } from "react";
import {
  AreaChart,
  Area,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
} from "recharts";
import { ipc } from "../services/ipc";
import { TrendingUp } from "lucide-react";

interface MetricsSnapshot {
  timestamp: string;
  activeViewers: number;
  totalWorkers: number;
  segments: number;
  bytesReceived: number;
  heartbeats: number;
  adsWatched: number;
}

interface SessionChartProps {
  sessionId: number;
}

export default function SessionChart({ sessionId }: SessionChartProps) {
  const [data, setData] = useState<MetricsSnapshot[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    ipc
      .call<MetricsSnapshot[]>("sessions.timeline", { sessionId })
      .then((snapshots) => {
        setData(
          (snapshots || []).map((s) => ({
            ...s,
            time: new Date(s.timestamp).toLocaleTimeString("en-US", {
              hour: "2-digit",
              minute: "2-digit",
              hour12: false,
            }),
          })),
        );
      })
      .catch(() => {})
      .finally(() => setLoading(false));
  }, [sessionId]);

  if (loading) {
    return (
      <div className="h-48 flex items-center justify-center text-gray-600 text-sm">
        Loading timeline...
      </div>
    );
  }

  if (data.length < 2) {
    return (
      <div className="h-48 flex items-center justify-center text-gray-600 text-sm">
        Not enough data points for chart
      </div>
    );
  }

  return (
    <div className="space-y-2">
      <div className="flex items-center gap-2 text-sm text-gray-400">
        <TrendingUp size={14} />
        <span>Viewers Timeline (Session #{sessionId})</span>
      </div>
      <div className="h-48">
        <ResponsiveContainer width="100%" height="100%">
          <AreaChart
            data={data}
            margin={{ top: 5, right: 5, left: -20, bottom: 0 }}
          >
            <defs>
              <linearGradient id={`grad-${sessionId}`} x1="0" y1="0" x2="0" y2="1">
                <stop offset="5%" stopColor="#3b82f6" stopOpacity={0.3} />
                <stop offset="95%" stopColor="#3b82f6" stopOpacity={0} />
              </linearGradient>
            </defs>
            <CartesianGrid strokeDasharray="3 3" stroke="#1f2937" />
            <XAxis
              dataKey="time"
              stroke="#4b5563"
              tick={{ fontSize: 10 }}
              interval="preserveStartEnd"
            />
            <YAxis stroke="#4b5563" tick={{ fontSize: 10 }} />
            <Tooltip
              contentStyle={{
                backgroundColor: "#111827",
                border: "1px solid #374151",
                borderRadius: "8px",
                fontSize: "12px",
              }}
            />
            <Area
              type="monotone"
              dataKey="activeViewers"
              stroke="#3b82f6"
              fill={`url(#grad-${sessionId})`}
              strokeWidth={2}
              dot={false}
              name="Viewers"
            />
          </AreaChart>
        </ResponsiveContainer>
      </div>
    </div>
  );
}
