import { useState, useRef, useEffect } from "react";
import { Download, Filter } from "lucide-react";
import { useLogs } from "../hooks/useIPC";
import type { LogEntry } from "../types";

const levelColors: Record<string, string> = {
  DEBUG: "text-gray-500",
  INFO: "text-blue-400",
  WARN: "text-amber-400",
  ERROR: "text-red-400",
};

const levelBadgeColors: Record<string, string> = {
  DEBUG: "bg-gray-800 text-gray-400",
  INFO: "bg-blue-500/15 text-blue-400",
  WARN: "bg-amber-500/15 text-amber-400",
  ERROR: "bg-red-500/15 text-red-400",
};

export default function LogViewer() {
  const logs = useLogs(1000);
  const [filter, setFilter] = useState("");
  const [levelFilter, setLevelFilter] = useState<string>("all");
  const [autoScroll, setAutoScroll] = useState(true);
  const scrollRef = useRef<HTMLDivElement>(null);

  // Auto-scroll to bottom when new logs arrive
  useEffect(() => {
    if (autoScroll && scrollRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight;
    }
  }, [logs, autoScroll]);

  const filteredLogs = logs.filter((log) => {
    if (levelFilter !== "all" && log.level !== levelFilter) return false;
    if (filter && !log.message.toLowerCase().includes(filter.toLowerCase())) return false;
    return true;
  });

  const formatTimestamp = (ts: string) => {
    try {
      return new Date(ts).toLocaleTimeString("en-US", { hour12: false });
    } catch {
      return ts;
    }
  };

  return (
    <div className="h-full flex flex-col p-6 gap-4">
      {/* Header */}
      <div className="flex items-center justify-between shrink-0">
        <div>
          <h1 className="text-2xl font-bold text-gray-100">Logs</h1>
          <p className="text-sm text-gray-500 mt-1">
            {filteredLogs.length} entries
            {filter || levelFilter !== "all" ? " (filtered)" : ""}
          </p>
        </div>
        <div className="flex items-center gap-2">
          <button
            className={`btn-ghost text-sm ${autoScroll ? "text-blue-400" : ""}`}
            onClick={() => setAutoScroll(!autoScroll)}
          >
            Auto-scroll: {autoScroll ? "ON" : "OFF"}
          </button>
          <button className="btn-ghost flex items-center gap-1">
            <Download size={14} />
            Export
          </button>
        </div>
      </div>

      {/* Filters */}
      <div className="flex items-center gap-3 shrink-0">
        <div className="relative flex-1">
          <Filter size={14} className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-500" />
          <input
            type="text"
            className="input-field pl-9 text-sm"
            placeholder="Filter logs..."
            value={filter}
            onChange={(e) => setFilter(e.target.value)}
          />
        </div>
        <div className="flex items-center gap-1 bg-gray-900 rounded-lg p-1">
          {["all", "DEBUG", "INFO", "WARN", "ERROR"].map((level) => (
            <button
              key={level}
              className={`px-3 py-1 rounded-md text-xs font-medium transition-colors ${
                levelFilter === level
                  ? "bg-gray-700 text-gray-100"
                  : "text-gray-500 hover:text-gray-300"
              }`}
              onClick={() => setLevelFilter(level)}
            >
              {level === "all" ? "All" : level}
            </button>
          ))}
        </div>
      </div>

      {/* Log Entries */}
      <div
        ref={scrollRef}
        className="flex-1 overflow-y-auto bg-gray-900/50 rounded-xl border border-gray-800 font-mono text-xs"
      >
        {filteredLogs.length === 0 ? (
          <div className="flex items-center justify-center h-full text-gray-600">
            No log entries
          </div>
        ) : (
          <div className="p-2 space-y-px">
            {filteredLogs.map((log, i) => (
              <LogLine key={i} log={log} formatTimestamp={formatTimestamp} />
            ))}
          </div>
        )}
      </div>
    </div>
  );
}

interface LogLineProps {
  log: LogEntry;
  formatTimestamp: (ts: string) => string;
}

function LogLine({ log, formatTimestamp }: LogLineProps) {
  return (
    <div className="flex items-start gap-2 px-2 py-1 hover:bg-gray-800/50 rounded group">
      <span className="text-gray-600 shrink-0 w-20">
        {formatTimestamp(log.timestamp)}
      </span>
      <span
        className={`shrink-0 w-12 text-center rounded px-1 ${
          levelBadgeColors[log.level] || "text-gray-400"
        }`}
      >
        {log.level}
      </span>
      {log.component && (
        <span className="text-cyan-600 shrink-0">[{log.component}]</span>
      )}
      {log.worker && (
        <span className="text-indigo-600 shrink-0">[{log.worker}]</span>
      )}
      <span className={levelColors[log.level] || "text-gray-400"}>
        {log.message}
      </span>
    </div>
  );
}
