import { useState, useEffect } from "react";
import {
  Zap,
  Github,
  Cpu,
  Globe,
  Database,
  Activity,
  Shield,
  Package,
  ExternalLink,
} from "lucide-react";
import { ipc } from "../services/ipc";

interface SessionStats {
  totalSessions: number;
  totalSegments: number;
  totalBytes: number;
  totalAds: number;
  peakViewers: number;
}

function formatBytes(b: number): string {
  if (b < 1073741824) return `${(b / 1048576).toFixed(1)} MB`;
  return `${(b / 1073741824).toFixed(2)} GB`;
}

export default function AboutPage() {
  const [stats, setStats] = useState<SessionStats | null>(null);

  useEffect(() => {
    ipc.call<SessionStats>("sessions.stats").then(setStats).catch(() => {});
  }, []);

  return (
    <div className="h-full overflow-y-auto p-6 space-y-6">
      {/* Header with logo */}
      <div className="card flex items-center gap-6 py-6">
        <div className="w-16 h-16 rounded-2xl bg-blue-600/20 flex items-center justify-center">
          <Zap size={32} className="text-blue-400" />
        </div>
        <div>
          <h1 className="text-3xl font-bold text-gray-100">MongeBot</h1>
          <p className="text-sm text-gray-500 mt-1">
            Modular Multi-Platform Viewer Bot
          </p>
          <div className="flex items-center gap-2 mt-2">
            <span className="badge-info">v2.0.0</span>
            <span className="badge bg-gray-800 text-gray-400">Go 1.26</span>
            <span className="badge bg-gray-800 text-gray-400">Tauri 2.0</span>
          </div>
        </div>
      </div>

      {/* Tech Stack */}
      <div className="card space-y-4">
        <h2 className="text-lg font-semibold text-gray-200 flex items-center gap-2">
          <Package size={18} className="text-emerald-400" />
          Tech Stack
        </h2>

        <div className="grid grid-cols-2 gap-3">
          <TechItem name="Backend" value="Go 1.26" icon={<Cpu size={14} className="text-cyan-400" />} detail="Green Tea GC, goroutines, Swiss Tables" />
          <TechItem name="Frontend" value="React + TypeScript" icon={<Globe size={14} className="text-blue-400" />} detail="Vite, TailwindCSS 4, Recharts" />
          <TechItem name="Desktop" value="Tauri 2.0" icon={<Shield size={14} className="text-orange-400" />} detail="Rust WebView, ~2MB bundle" />
          <TechItem name="Database" value="SQLite (Pure Go)" icon={<Database size={14} className="text-amber-400" />} detail="WAL mode, modernc.org/sqlite" />
          <TechItem name="WebSocket" value="coder/websocket" icon={<Activity size={14} className="text-violet-400" />} detail="Context-aware, concurrent-safe" />
          <TechItem name="Crypto" value="AES-256-GCM" icon={<Shield size={14} className="text-red-400" />} detail="PBKDF2 key derivation (600k iter)" />
        </div>
      </div>

      {/* Platforms */}
      <div className="card space-y-4">
        <h2 className="text-lg font-semibold text-gray-200 flex items-center gap-2">
          <Globe size={18} className="text-blue-400" />
          Supported Platforms
        </h2>

        <div className="grid grid-cols-3 gap-3">
          <PlatformCard name="Twitch" status="Full" features="GQL, HLS, Spade, PubSub, Chat, Ads" color="text-purple-400" />
          <PlatformCard name="Kick" status="Basic" features="API, HLS, Chat (Pusher)" color="text-green-400" />
          <PlatformCard name="YouTube" status="Stub" features="Innertube API, Live detection" color="text-red-400" />
        </div>
      </div>

      {/* Lifetime Stats */}
      {stats && (
        <div className="card space-y-4">
          <h2 className="text-lg font-semibold text-gray-200 flex items-center gap-2">
            <Activity size={18} className="text-amber-400" />
            Lifetime Statistics
          </h2>

          <div className="grid grid-cols-5 gap-4">
            <StatItem label="Total Sessions" value={stats.totalSessions.toString()} />
            <StatItem label="Peak Viewers" value={stats.peakViewers.toString()} />
            <StatItem label="Segments" value={stats.totalSegments.toLocaleString()} />
            <StatItem label="Data Transferred" value={formatBytes(stats.totalBytes)} />
            <StatItem label="Ads Watched" value={stats.totalAds.toString()} />
          </div>
        </div>
      )}

      {/* Features */}
      <div className="card space-y-4">
        <h2 className="text-lg font-semibold text-gray-200">Features</h2>
        <div className="grid grid-cols-2 gap-x-8 gap-y-1 text-sm text-gray-400">
          <Feature text="Multi-platform plugin architecture" />
          <Feature text="Multi-account profile system" />
          <Feature text="Multi-channel simultaneous" />
          <Feature text="Real-time Recharts dashboard" />
          <Feature text="4 proxy rotation strategies" />
          <Feature text="Concurrent proxy health checker" />
          <Feature text="Auto proxy scraper (3 APIs)" />
          <Feature text="Encrypted token vault (AES-256)" />
          <Feature text="Token cookie import (4 formats)" />
          <Feature text="TLS fingerprint rotation" />
          <Feature text="Circuit breaker pattern" />
          <Feature text="Exponential backoff with jitter" />
          <Feature text="Stream auto-detect scheduler" />
          <Feature text="FFmpeg restream (5 presets)" />
          <Feature text="SQLite metrics persistence" />
          <Feature text="Session history with charts" />
          <Feature text="Toast notifications system" />
          <Feature text="Dark/light theme + 6 accents" />
          <Feature text="Keyboard shortcuts (Ctrl+1-8)" />
          <Feature text="Config export/import encrypted" />
          <Feature text="Docker + CI/CD ready" />
          <Feature text="Cross-platform (Win/Mac/Linux)" />
        </div>
      </div>

      {/* Footer */}
      <div className="text-center py-4 text-xs text-gray-700">
        Built with Go, React, Tauri, and Rust
      </div>
    </div>
  );
}

function TechItem({ name, value, icon, detail }: { name: string; value: string; icon: React.ReactNode; detail: string }) {
  return (
    <div className="flex items-start gap-3 p-3 rounded-lg bg-gray-800/30">
      <span className="mt-0.5">{icon}</span>
      <div>
        <p className="text-sm font-medium text-gray-200">{value}</p>
        <p className="text-[10px] text-gray-600">{name} — {detail}</p>
      </div>
    </div>
  );
}

function PlatformCard({ name, status, features, color }: { name: string; status: string; features: string; color: string }) {
  return (
    <div className="p-3 rounded-xl border border-gray-800 bg-gray-800/30">
      <div className="flex items-center justify-between mb-1">
        <span className={`text-sm font-semibold ${color}`}>{name}</span>
        <span className="badge-info text-[10px]">{status}</span>
      </div>
      <p className="text-[10px] text-gray-500">{features}</p>
    </div>
  );
}

function StatItem({ label, value }: { label: string; value: string }) {
  return (
    <div className="text-center">
      <p className="text-xl font-bold text-gray-200">{value}</p>
      <p className="text-[10px] text-gray-600 mt-0.5">{label}</p>
    </div>
  );
}

function Feature({ text }: { text: string }) {
  return (
    <div className="flex items-center gap-2 py-1">
      <div className="w-1.5 h-1.5 rounded-full bg-emerald-500/60" />
      <span>{text}</span>
    </div>
  );
}
