import { useEffect, useRef, useState } from "react";
import {
  AreaChart,
  Area,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  BarChart,
  Bar,
} from "recharts";
import type { EngineMetrics } from "../types";

interface DataPoint {
  time: string;
  viewers: number;
  segments: number;
  bytes: number;
  heartbeats: number;
}

interface MetricsChartProps {
  metrics: EngineMetrics | null;
  maxPoints?: number;
}

export function ViewersChart({ metrics, maxPoints = 60 }: MetricsChartProps) {
  const [data, setData] = useState<DataPoint[]>([]);
  const prevBytes = useRef(0);

  useEffect(() => {
    if (!metrics) return;

    const now = new Date();
    const time = `${now.getMinutes().toString().padStart(2, "0")}:${now.getSeconds().toString().padStart(2, "0")}`;

    const bytesPerSec = metrics.bytesReceived - prevBytes.current;
    prevBytes.current = metrics.bytesReceived;

    setData((prev) => {
      const next = [
        ...prev,
        {
          time,
          viewers: metrics.activeViewers,
          segments: metrics.segmentsFetched,
          bytes: Math.max(0, bytesPerSec),
          heartbeats: metrics.heartbeatsSent,
        },
      ];
      return next.slice(-maxPoints);
    });
  }, [metrics, maxPoints]);

  if (data.length < 2) {
    return (
      <div className="h-full flex items-center justify-center text-gray-600 text-sm">
        Collecting data...
      </div>
    );
  }

  return (
    <ResponsiveContainer width="100%" height="100%">
      <AreaChart data={data} margin={{ top: 5, right: 5, left: -20, bottom: 0 }}>
        <defs>
          <linearGradient id="viewerGrad" x1="0" y1="0" x2="0" y2="1">
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
          labelStyle={{ color: "#9ca3af" }}
        />
        <Area
          type="monotone"
          dataKey="viewers"
          stroke="#3b82f6"
          fill="url(#viewerGrad)"
          strokeWidth={2}
          dot={false}
          name="Active Viewers"
        />
      </AreaChart>
    </ResponsiveContainer>
  );
}

export function BandwidthChart({ metrics, maxPoints = 60 }: MetricsChartProps) {
  const [data, setData] = useState<{ time: string; kbps: number }[]>([]);
  const prevBytes = useRef(0);

  useEffect(() => {
    if (!metrics) return;

    const now = new Date();
    const time = `${now.getMinutes().toString().padStart(2, "0")}:${now.getSeconds().toString().padStart(2, "0")}`;

    const delta = metrics.bytesReceived - prevBytes.current;
    prevBytes.current = metrics.bytesReceived;

    // Convert to KB/s (assuming 5s interval from metrics loop)
    const kbps = Math.max(0, delta / 1024 / 5);

    setData((prev) => [...prev, { time, kbps }].slice(-maxPoints));
  }, [metrics, maxPoints]);

  if (data.length < 2) {
    return (
      <div className="h-full flex items-center justify-center text-gray-600 text-sm">
        Collecting data...
      </div>
    );
  }

  return (
    <ResponsiveContainer width="100%" height="100%">
      <BarChart data={data} margin={{ top: 5, right: 5, left: -20, bottom: 0 }}>
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
          formatter={(value) => [`${Number(value).toFixed(1)} KB/s`, "Bandwidth"]}
        />
        <Bar dataKey="kbps" fill="#10b981" radius={[2, 2, 0, 0]} name="KB/s" />
      </BarChart>
    </ResponsiveContainer>
  );
}
