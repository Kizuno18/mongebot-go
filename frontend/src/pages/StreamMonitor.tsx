import { useState } from "react";
import {
  Radio,
  Play,
  Square,
  MonitorPlay,
  Tv,
  Signal,
  Users,
  Gamepad2,
  Clock,
} from "lucide-react";

interface StreamInfo {
  channel: string;
  status: "online" | "offline" | "unknown";
  title?: string;
  game?: string;
  viewers?: number;
  startedAt?: string;
}

// Quality presets matching Go backend
const qualityPresets = [
  { key: "potato", name: "Potato", res: "320x180", bitrate: "100k", fps: 10 },
  { key: "low", name: "Low", res: "640x360", bitrate: "800k", fps: 24 },
  { key: "medium", name: "Medium", res: "1280x720", bitrate: "2500k", fps: 30 },
  { key: "high", name: "High", res: "1920x1080", bitrate: "4500k", fps: 30 },
  { key: "ultra", name: "Ultra", res: "1920x1080", bitrate: "6000k", fps: 60 },
];

export default function StreamMonitor() {
  const [streamInfo] = useState<StreamInfo>({
    channel: "",
    status: "unknown",
  });
  const [checkChannel, setCheckChannel] = useState("");
  const [selectedPreset, setSelectedPreset] = useState("medium");
  const [streamKey, setStreamKey] = useState("");
  const [inputFile, setInputFile] = useState("video.mp4");
  const [isRestreaming, setIsRestreaming] = useState(false);

  const isOnline = streamInfo.status === "online";

  return (
    <div className="h-full overflow-y-auto p-6 space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-100">Stream Monitor</h1>
          <p className="text-sm text-gray-500 mt-1">
            Monitor streams and manage FFmpeg restreaming
          </p>
        </div>
      </div>

      {/* Stream Status Checker */}
      <div className="card space-y-4">
        <div className="flex items-center gap-2 mb-1">
          <Signal size={18} className="text-blue-400" />
          <h2 className="text-lg font-semibold text-gray-200">
            Stream Status
          </h2>
        </div>

        <div className="flex items-center gap-3">
          <input
            type="text"
            className="input-field flex-1"
            placeholder="Enter channel name to check..."
            value={checkChannel}
            onChange={(e) => setCheckChannel(e.target.value)}
          />
          <button className="btn-primary flex items-center gap-2">
            <Radio size={16} />
            Check
          </button>
        </div>

        {streamInfo.channel && (
          <div className="bg-gray-800/50 rounded-xl p-4 space-y-3">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-3">
                <div
                  className={`w-3 h-3 rounded-full ${
                    isOnline ? "bg-red-500 animate-pulse" : "bg-gray-600"
                  }`}
                />
                <span className="text-lg font-semibold text-gray-100">
                  {streamInfo.channel}
                </span>
              </div>
              <span
                className={isOnline ? "badge-danger" : "bg-gray-800 text-gray-400 badge"}
              >
                {isOnline ? "LIVE" : "OFFLINE"}
              </span>
            </div>

            {isOnline && (
              <div className="grid grid-cols-2 gap-4 mt-2">
                <InfoItem icon={<Tv size={14} />} label="Title" value={streamInfo.title || "—"} />
                <InfoItem icon={<Gamepad2 size={14} />} label="Game" value={streamInfo.game || "—"} />
                <InfoItem
                  icon={<Users size={14} />}
                  label="Viewers"
                  value={streamInfo.viewers?.toLocaleString() || "—"}
                />
                <InfoItem icon={<Clock size={14} />} label="Started" value={streamInfo.startedAt || "—"} />
              </div>
            )}
          </div>
        )}
      </div>

      {/* Restream Controls */}
      <div className="card space-y-4">
        <div className="flex items-center gap-2 mb-1">
          <MonitorPlay size={18} className="text-emerald-400" />
          <h2 className="text-lg font-semibold text-gray-200">
            FFmpeg Restream
          </h2>
        </div>

        <div className="grid grid-cols-2 gap-4">
          <div>
            <label className="text-xs text-gray-500 mb-1 block">
              Input File
            </label>
            <input
              type="text"
              className="input-field font-mono"
              value={inputFile}
              onChange={(e) => setInputFile(e.target.value)}
              placeholder="video.mp4"
            />
          </div>
          <div>
            <label className="text-xs text-gray-500 mb-1 block">
              Stream Key
            </label>
            <input
              type="password"
              className="input-field font-mono"
              value={streamKey}
              onChange={(e) => setStreamKey(e.target.value)}
              placeholder="live_123456789_xxxxx"
            />
          </div>
        </div>

        {/* Quality Presets */}
        <div>
          <label className="text-xs text-gray-500 mb-2 block">
            Quality Preset
          </label>
          <div className="grid grid-cols-5 gap-2">
            {qualityPresets.map((preset) => (
              <button
                key={preset.key}
                className={`
                  p-3 rounded-xl border text-center transition-all text-sm
                  ${
                    selectedPreset === preset.key
                      ? "border-emerald-500/40 bg-emerald-500/10 text-emerald-300"
                      : "border-gray-800 bg-gray-800/30 text-gray-400 hover:border-gray-700"
                  }
                `}
                onClick={() => setSelectedPreset(preset.key)}
              >
                <p className="font-semibold">{preset.name}</p>
                <p className="text-[10px] mt-1 opacity-60">{preset.res}</p>
                <p className="text-[10px] opacity-60">
                  {preset.bitrate} · {preset.fps}fps
                </p>
              </button>
            ))}
          </div>
        </div>

        {/* Start/Stop */}
        <div className="flex justify-end">
          {!isRestreaming ? (
            <button
              className="btn-primary flex items-center gap-2"
              disabled={!streamKey}
              onClick={() => setIsRestreaming(true)}
            >
              <Play size={16} />
              Start Restream
            </button>
          ) : (
            <button
              className="btn-danger flex items-center gap-2"
              onClick={() => setIsRestreaming(false)}
            >
              <Square size={16} />
              Stop Restream
            </button>
          )}
        </div>
      </div>
    </div>
  );
}

interface InfoItemProps {
  icon: React.ReactNode;
  label: string;
  value: string;
}

function InfoItem({ icon, label, value }: InfoItemProps) {
  return (
    <div className="flex items-start gap-2">
      <span className="text-gray-500 mt-0.5">{icon}</span>
      <div>
        <p className="text-[10px] uppercase text-gray-600">{label}</p>
        <p className="text-sm text-gray-300 truncate">{value}</p>
      </div>
    </div>
  );
}
