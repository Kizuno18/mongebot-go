import { useState, useEffect } from "react";
import {
  Save,
  RotateCcw,
  Sliders,
  Cpu,
  Shield,
  Timer,
  ToggleLeft,
  ToggleRight,
  Palette,
  Download,
  Upload,
  Sun,
  Moon,
} from "lucide-react";
import { ipc } from "../services/ipc";
import type { AppConfig } from "../types";
import { loadPrefs, savePrefs, applyTheme, availableAccents, type Theme, type AccentColor } from "../stores/theme";

export default function SettingsPage() {
  const [config, setConfig] = useState<AppConfig | null>(null);
  const [dirty, setDirty] = useState(false);

  useEffect(() => {
    ipc.call<AppConfig>("config.get").then(setConfig).catch(() => {});
  }, []);

  const updateEngine = (key: string, value: unknown) => {
    if (!config) return;
    setConfig({
      ...config,
      engine: { ...config.engine, [key]: value },
    });
    setDirty(true);
  };

  const toggleFeature = (feature: string) => {
    if (!config) return;
    setConfig({
      ...config,
      engine: {
        ...config.engine,
        features: {
          ...config.engine.features,
          [feature]: !config.engine.features[feature as keyof typeof config.engine.features],
        },
      },
    });
    setDirty(true);
  };

  const handleSave = async () => {
    if (!config) return;
    try {
      await ipc.call("config.set", { engine: config.engine });
      setDirty(false);
    } catch (err) {
      alert(`Save failed: ${err}`);
    }
  };

  const features = config?.engine.features;

  return (
    <div className="h-full overflow-y-auto p-6 space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-100">Settings</h1>
          <p className="text-sm text-gray-500 mt-1">
            Engine configuration and feature toggles
          </p>
        </div>
        <div className="flex items-center gap-2">
          {dirty && (
            <button className="btn-primary flex items-center gap-2" onClick={handleSave}>
              <Save size={16} />
              Save Changes
            </button>
          )}
        </div>
      </div>

      {/* Engine Settings */}
      <div className="card space-y-4">
        <div className="flex items-center gap-2 mb-2">
          <Cpu size={18} className="text-blue-400" />
          <h2 className="text-lg font-semibold text-gray-200">Engine</h2>
        </div>

        <div className="grid grid-cols-2 gap-4">
          <SettingField
            label="Max Workers"
            description="Maximum concurrent viewer connections"
          >
            <input
              type="number"
              className="input-field w-24"
              value={config?.engine.maxWorkers ?? 50}
              onChange={(e) => updateEngine("maxWorkers", parseInt(e.target.value))}
              min={1}
              max={1000}
            />
          </SettingField>

          <SettingField
            label="Max Retries"
            description="Retry attempts per viewer before giving up"
          >
            <input
              type="number"
              className="input-field w-24"
              value={config?.engine.maxRetries ?? 3}
              onChange={(e) => updateEngine("maxRetries", parseInt(e.target.value))}
              min={0}
              max={10}
            />
          </SettingField>

          <SettingField
            label="Proxy Timeout"
            description="Connection timeout for proxy connections"
          >
            <input
              type="text"
              className="input-field w-24"
              value={config?.engine.proxyTimeout ?? "60s"}
              onChange={(e) => updateEngine("proxyTimeout", e.target.value)}
              placeholder="60s"
            />
          </SettingField>

          <SettingField
            label="Restart Interval"
            description="How often to check for dead workers"
          >
            <input
              type="text"
              className="input-field w-24"
              value={config?.engine.restartInterval ?? "10s"}
              onChange={(e) => updateEngine("restartInterval", e.target.value)}
              placeholder="10s"
            />
          </SettingField>
        </div>
      </div>

      {/* Feature Toggles */}
      <div className="card space-y-4">
        <div className="flex items-center gap-2 mb-2">
          <Sliders size={18} className="text-emerald-400" />
          <h2 className="text-lg font-semibold text-gray-200">
            Feature Toggles
          </h2>
        </div>
        <p className="text-xs text-gray-500">
          Enable or disable individual viewer behaviors
        </p>

        <div className="grid grid-cols-2 gap-3">
          <FeatureToggle
            label="Spade Analytics"
            description="Send video-play and minute-watched events"
            enabled={features?.spade ?? true}
            onToggle={() => toggleFeature("spade")}
          />
          <FeatureToggle
            label="HLS Segments"
            description="Fetch video segments to simulate playback"
            enabled={features?.segments ?? true}
            onToggle={() => toggleFeature("segments")}
          />
          <FeatureToggle
            label="GQL Heartbeat"
            description="Send WatchTrackQuery pulses periodically"
            enabled={features?.gqlPulse ?? true}
            onToggle={() => toggleFeature("gqlPulse")}
          />
          <FeatureToggle
            label="PubSub"
            description="Connect to Twitch PubSub for stream events"
            enabled={features?.pubsub ?? true}
            onToggle={() => toggleFeature("pubsub")}
          />
          <FeatureToggle
            label="IRC Chat"
            description="Join channel chat to appear as active viewer"
            enabled={features?.chat ?? true}
            onToggle={() => toggleFeature("chat")}
          />
          <FeatureToggle
            label="Watch Ads"
            description="Detect and simulate watching commercial ads"
            enabled={features?.ads ?? true}
            onToggle={() => toggleFeature("ads")}
          />
        </div>
      </div>

      {/* Log Settings */}
      <div className="card space-y-4">
        <div className="flex items-center gap-2 mb-2">
          <Timer size={18} className="text-amber-400" />
          <h2 className="text-lg font-semibold text-gray-200">Logging</h2>
        </div>

        <div className="grid grid-cols-2 gap-4">
          <SettingField label="Log Level" description="Minimum log level to capture">
            <select
              className="input-field w-32"
              value={config?.logging.level ?? "info"}
              onChange={() => setDirty(true)}
            >
              <option value="debug">Debug</option>
              <option value="info">Info</option>
              <option value="warn">Warning</option>
              <option value="error">Error</option>
            </select>
          </SettingField>

          <SettingField
            label="Ring Buffer Size"
            description="Number of log entries kept in memory for UI"
          >
            <input
              type="number"
              className="input-field w-24"
              value={config?.logging.ringBuffer ?? 1000}
              min={100}
              max={10000}
            />
          </SettingField>
        </div>
      </div>

      {/* Appearance */}
      <ThemeSection />

      {/* Data Management */}
      <div className="card space-y-4">
        <div className="flex items-center gap-2 mb-2">
          <Download size={18} className="text-cyan-400" />
          <h2 className="text-lg font-semibold text-gray-200">Data Management</h2>
        </div>

        <div className="grid grid-cols-2 gap-3">
          <button className="p-4 rounded-xl border border-gray-800 bg-gray-800/30 hover:border-gray-700 transition-all text-left">
            <div className="flex items-center gap-2 mb-1">
              <Download size={16} className="text-blue-400" />
              <span className="text-sm font-medium text-gray-200">Export Config</span>
            </div>
            <p className="text-[10px] text-gray-500">
              Download profiles, settings, and proxy list as encrypted archive
            </p>
          </button>

          <button className="p-4 rounded-xl border border-gray-800 bg-gray-800/30 hover:border-gray-700 transition-all text-left">
            <div className="flex items-center gap-2 mb-1">
              <Upload size={16} className="text-emerald-400" />
              <span className="text-sm font-medium text-gray-200">Import Config</span>
            </div>
            <p className="text-[10px] text-gray-500">
              Restore configuration from a previously exported archive
            </p>
          </button>
        </div>
      </div>
    </div>
  );
}

interface SettingFieldProps {
  label: string;
  description: string;
  children: React.ReactNode;
}

function SettingField({ label, description, children }: SettingFieldProps) {
  return (
    <div className="flex items-center justify-between p-3 rounded-lg bg-gray-800/30">
      <div>
        <p className="text-sm font-medium text-gray-200">{label}</p>
        <p className="text-xs text-gray-500 mt-0.5">{description}</p>
      </div>
      {children}
    </div>
  );
}

interface FeatureToggleProps {
  label: string;
  description: string;
  enabled: boolean;
  onToggle: () => void;
}

function FeatureToggle({ label, description, enabled, onToggle }: FeatureToggleProps) {
  return (
    <button
      onClick={onToggle}
      className={`
        flex items-center justify-between p-3 rounded-lg transition-colors text-left
        ${enabled ? "bg-emerald-500/10 border border-emerald-500/20" : "bg-gray-800/30 border border-gray-800"}
      `}
    >
      <div>
        <p className="text-sm font-medium text-gray-200">{label}</p>
        <p className="text-xs text-gray-500 mt-0.5">{description}</p>
      </div>
      {enabled ? (
        <ToggleRight size={24} className="text-emerald-400 shrink-0" />
      ) : (
        <ToggleLeft size={24} className="text-gray-600 shrink-0" />
      )}
    </button>
  );
}

function ThemeSection() {
  const [prefs, setPrefs] = useState(() => loadPrefs());

  useEffect(() => {
    applyTheme(prefs);
  }, [prefs]);

  const updatePref = <K extends keyof ReturnType<typeof loadPrefs>>(
    key: K,
    value: ReturnType<typeof loadPrefs>[K],
  ) => {
    const next = { ...prefs, [key]: value };
    setPrefs(next);
    savePrefs(next);
  };

  return (
    <div className="card space-y-4">
      <div className="flex items-center gap-2 mb-2">
        <Palette size={18} className="text-violet-400" />
        <h2 className="text-lg font-semibold text-gray-200">Appearance</h2>
      </div>

      {/* Theme toggle */}
      <div className="flex items-center justify-between p-3 rounded-lg bg-gray-800/30">
        <div>
          <p className="text-sm font-medium text-gray-200">Theme</p>
          <p className="text-xs text-gray-500 mt-0.5">Switch between dark and light mode</p>
        </div>
        <div className="flex items-center gap-1 bg-gray-900 rounded-lg p-1">
          <button
            className={`px-3 py-1.5 rounded-md flex items-center gap-1.5 text-xs transition-colors ${
              prefs.theme === "dark" ? "bg-gray-700 text-gray-100" : "text-gray-500"
            }`}
            onClick={() => updatePref("theme", "dark")}
          >
            <Moon size={12} />
            Dark
          </button>
          <button
            className={`px-3 py-1.5 rounded-md flex items-center gap-1.5 text-xs transition-colors ${
              prefs.theme === "light" ? "bg-gray-700 text-gray-100" : "text-gray-500"
            }`}
            onClick={() => updatePref("theme", "light")}
          >
            <Sun size={12} />
            Light
          </button>
        </div>
      </div>

      {/* Accent color */}
      <div className="flex items-center justify-between p-3 rounded-lg bg-gray-800/30">
        <div>
          <p className="text-sm font-medium text-gray-200">Accent Color</p>
          <p className="text-xs text-gray-500 mt-0.5">Primary color for buttons and highlights</p>
        </div>
        <div className="flex items-center gap-1.5">
          {availableAccents.map((accent) => (
            <button
              key={accent.key}
              className={`w-6 h-6 rounded-full ${accent.swatch} transition-all ${
                prefs.accentColor === accent.key
                  ? "ring-2 ring-white/30 scale-110"
                  : "opacity-60 hover:opacity-100"
              }`}
              title={accent.label}
              onClick={() => updatePref("accentColor", accent.key)}
            />
          ))}
        </div>
      </div>

      {/* Compact mode */}
      <div className="flex items-center justify-between p-3 rounded-lg bg-gray-800/30">
        <div>
          <p className="text-sm font-medium text-gray-200">Compact Mode</p>
          <p className="text-xs text-gray-500 mt-0.5">Reduce spacing for smaller screens</p>
        </div>
        <button onClick={() => updatePref("compactMode", !prefs.compactMode)}>
          {prefs.compactMode ? (
            <ToggleRight size={24} className="text-emerald-400" />
          ) : (
            <ToggleLeft size={24} className="text-gray-600" />
          )}
        </button>
      </div>

      {/* Show charts */}
      <div className="flex items-center justify-between p-3 rounded-lg bg-gray-800/30">
        <div>
          <p className="text-sm font-medium text-gray-200">Show Charts</p>
          <p className="text-xs text-gray-500 mt-0.5">Display real-time graphs on Dashboard</p>
        </div>
        <button onClick={() => updatePref("showCharts", !prefs.showCharts)}>
          {prefs.showCharts ? (
            <ToggleRight size={24} className="text-emerald-400" />
          ) : (
            <ToggleLeft size={24} className="text-gray-600" />
          )}
        </button>
      </div>
    </div>
  );
}
