import { useState } from "react";
import {
  CalendarClock,
  Plus,
  Trash2,
  Radio,
  Clock,
  ToggleRight,
  ToggleLeft,
  Play,
} from "lucide-react";

interface ScheduleRule {
  id: string;
  name: string;
  channel: string;
  trigger: "stream_live" | "scheduled" | "manual";
  workers: number;
  enabled: boolean;
  startTime?: string;
  stopTime?: string;
  weekdays?: number[];
  maxDuration?: string;
}

const weekdayNames = ["Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"];

const triggerLabels: Record<string, { label: string; icon: React.ReactNode; desc: string }> = {
  stream_live: {
    label: "Stream Live",
    icon: <Radio size={14} className="text-red-400" />,
    desc: "Auto-start when streamer goes live",
  },
  scheduled: {
    label: "Time Schedule",
    icon: <Clock size={14} className="text-blue-400" />,
    desc: "Start/stop at specific times",
  },
  manual: {
    label: "Manual Only",
    icon: <Play size={14} className="text-gray-400" />,
    desc: "No auto-start, manual control only",
  },
};

export default function SchedulerPage() {
  const [rules, setRules] = useState<ScheduleRule[]>([
    {
      id: "1",
      name: "Main Stream Watch",
      channel: "streamer_name",
      trigger: "stream_live",
      workers: 50,
      enabled: true,
      maxDuration: "8h",
    },
    {
      id: "2",
      name: "Evening Schedule",
      channel: "another_streamer",
      trigger: "scheduled",
      workers: 25,
      enabled: false,
      startTime: "18:00",
      stopTime: "23:00",
      weekdays: [1, 2, 3, 4, 5],
    },
  ]);
  const [showCreate, setShowCreate] = useState(false);
  const [newRule, setNewRule] = useState<Partial<ScheduleRule>>({
    trigger: "stream_live",
    workers: 50,
    enabled: true,
  });

  const toggleRule = (id: string) => {
    setRules(rules.map((r) => (r.id === id ? { ...r, enabled: !r.enabled } : r)));
  };

  const deleteRule = (id: string) => {
    setRules(rules.filter((r) => r.id !== id));
  };

  const handleCreate = () => {
    if (!newRule.name || !newRule.channel) return;
    const rule: ScheduleRule = {
      id: `r${Date.now()}`,
      name: newRule.name || "",
      channel: newRule.channel || "",
      trigger: newRule.trigger || "stream_live",
      workers: newRule.workers || 50,
      enabled: true,
      startTime: newRule.startTime,
      stopTime: newRule.stopTime,
      weekdays: newRule.weekdays,
      maxDuration: newRule.maxDuration,
    };
    setRules([...rules, rule]);
    setShowCreate(false);
    setNewRule({ trigger: "stream_live", workers: 50, enabled: true });
  };

  return (
    <div className="h-full overflow-y-auto p-6 space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-100">Scheduler</h1>
          <p className="text-sm text-gray-500 mt-1">
            Automated start/stop rules for viewer sessions
          </p>
        </div>
        <button
          className="btn-primary flex items-center gap-2"
          onClick={() => setShowCreate(!showCreate)}
        >
          <Plus size={16} />
          New Rule
        </button>
      </div>

      {/* Create form */}
      {showCreate && (
        <div className="card space-y-4 animate-fade-in">
          <h3 className="text-sm font-semibold text-gray-300">Create Schedule Rule</h3>
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="text-xs text-gray-500 mb-1 block">Rule Name</label>
              <input
                className="input-field"
                placeholder="e.g., Main Stream Watch"
                value={newRule.name || ""}
                onChange={(e) => setNewRule({ ...newRule, name: e.target.value })}
              />
            </div>
            <div>
              <label className="text-xs text-gray-500 mb-1 block">Channel</label>
              <input
                className="input-field"
                placeholder="streamer_name"
                value={newRule.channel || ""}
                onChange={(e) => setNewRule({ ...newRule, channel: e.target.value })}
              />
            </div>
          </div>

          <div>
            <label className="text-xs text-gray-500 mb-2 block">Trigger Type</label>
            <div className="grid grid-cols-3 gap-2">
              {Object.entries(triggerLabels).map(([key, cfg]) => (
                <button
                  key={key}
                  className={`p-3 rounded-xl border text-left transition-all ${
                    newRule.trigger === key
                      ? "border-blue-500/40 bg-blue-500/10"
                      : "border-gray-800 bg-gray-800/30 hover:border-gray-700"
                  }`}
                  onClick={() => setNewRule({ ...newRule, trigger: key as ScheduleRule["trigger"] })}
                >
                  <div className="flex items-center gap-2 mb-1">
                    {cfg.icon}
                    <span className="text-sm font-medium text-gray-200">{cfg.label}</span>
                  </div>
                  <p className="text-[10px] text-gray-500">{cfg.desc}</p>
                </button>
              ))}
            </div>
          </div>

          {newRule.trigger === "scheduled" && (
            <div className="grid grid-cols-2 gap-4">
              <div>
                <label className="text-xs text-gray-500 mb-1 block">Start Time</label>
                <input
                  type="time"
                  className="input-field"
                  value={newRule.startTime || ""}
                  onChange={(e) => setNewRule({ ...newRule, startTime: e.target.value })}
                />
              </div>
              <div>
                <label className="text-xs text-gray-500 mb-1 block">Stop Time</label>
                <input
                  type="time"
                  className="input-field"
                  value={newRule.stopTime || ""}
                  onChange={(e) => setNewRule({ ...newRule, stopTime: e.target.value })}
                />
              </div>
            </div>
          )}

          <div className="flex justify-end gap-2">
            <button className="btn-ghost" onClick={() => setShowCreate(false)}>Cancel</button>
            <button className="btn-primary" onClick={handleCreate}>Create Rule</button>
          </div>
        </div>
      )}

      {/* Rules list */}
      <div className="space-y-3">
        {rules.map((rule) => (
          <div
            key={rule.id}
            className={`card flex items-center gap-4 py-3 transition-all ${
              rule.enabled ? "" : "opacity-50"
            }`}
          >
            <button onClick={() => toggleRule(rule.id)} className="shrink-0">
              {rule.enabled ? (
                <ToggleRight size={24} className="text-emerald-400" />
              ) : (
                <ToggleLeft size={24} className="text-gray-600" />
              )}
            </button>

            <div className="flex-1 min-w-0">
              <div className="flex items-center gap-2">
                <span className="text-sm font-semibold text-gray-200">{rule.name}</span>
                <span className="badge-info text-[10px]">#{rule.channel}</span>
              </div>
              <div className="flex items-center gap-2 mt-1 text-xs text-gray-500">
                {triggerLabels[rule.trigger]?.icon}
                <span>{triggerLabels[rule.trigger]?.label}</span>
                <span>·</span>
                <span>{rule.workers} workers</span>
                {rule.trigger === "scheduled" && rule.startTime && (
                  <>
                    <span>·</span>
                    <span>{rule.startTime} — {rule.stopTime}</span>
                  </>
                )}
                {rule.trigger === "scheduled" && rule.weekdays && (
                  <>
                    <span>·</span>
                    <span>{rule.weekdays.map((d) => weekdayNames[d]).join(", ")}</span>
                  </>
                )}
                {rule.maxDuration && (
                  <>
                    <span>·</span>
                    <span>max {rule.maxDuration}</span>
                  </>
                )}
              </div>
            </div>

            <button
              className="p-2 hover:bg-gray-800 rounded-lg transition-colors text-gray-600 hover:text-red-400"
              onClick={() => deleteRule(rule.id)}
            >
              <Trash2 size={14} />
            </button>
          </div>
        ))}

        {rules.length === 0 && (
          <div className="card flex flex-col items-center py-12 text-gray-600">
            <CalendarClock size={48} className="mb-3 opacity-30" />
            <p>No schedule rules yet</p>
          </div>
        )}
      </div>
    </div>
  );
}
