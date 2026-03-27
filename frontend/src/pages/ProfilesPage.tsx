import { useState } from "react";
import {
  Plus,
  Copy,
  Trash2,
  CheckCircle2,
  Circle,
  Edit3,
  Download,
  Upload,
  Tv,
  Users,
} from "lucide-react";
import type { ProfileConfig } from "../types";

// Placeholder profiles (will connect to IPC profile.* methods)
const mockProfiles: ProfileConfig[] = [
  {
    id: "p1",
    name: "Main Channel",
    platform: "twitch",
    channel: "streamer_name",
    active: true,
    maxWorkers: 50,
    features: {
      ads: true,
      chat: true,
      pubsub: true,
      segments: true,
      gqlPulse: true,
      spade: true,
    },
  },
  {
    id: "p2",
    name: "Alt Channel",
    platform: "twitch",
    channel: "another_streamer",
    active: false,
    maxWorkers: 25,
  },
];

export default function ProfilesPage() {
  const [profiles, setProfiles] = useState<ProfileConfig[]>(mockProfiles);
  const [showCreate, setShowCreate] = useState(false);
  const [newName, setNewName] = useState("");
  const [newChannel, setNewChannel] = useState("");

  const handleCreate = () => {
    if (!newName.trim() || !newChannel.trim()) return;

    const profile: ProfileConfig = {
      id: `p${Date.now()}`,
      name: newName,
      platform: "twitch",
      channel: newChannel,
      active: false,
    };

    setProfiles([...profiles, profile]);
    setNewName("");
    setNewChannel("");
    setShowCreate(false);
  };

  const handleActivate = (id: string) => {
    setProfiles(
      profiles.map((p) => ({ ...p, active: p.id === id })),
    );
  };

  const handleDelete = (id: string) => {
    setProfiles(profiles.filter((p) => p.id !== id));
  };

  const handleDuplicate = (profile: ProfileConfig) => {
    const clone: ProfileConfig = {
      ...profile,
      id: `p${Date.now()}`,
      name: `${profile.name} (Copy)`,
      active: false,
    };
    setProfiles([...profiles, clone]);
  };

  return (
    <div className="h-full overflow-y-auto p-6 space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-100">Profiles</h1>
          <p className="text-sm text-gray-500 mt-1">
            Multi-account management with per-channel configurations
          </p>
        </div>
        <div className="flex items-center gap-2">
          <button className="btn-ghost flex items-center gap-2 text-sm">
            <Download size={14} />
            Export
          </button>
          <button className="btn-ghost flex items-center gap-2 text-sm">
            <Upload size={14} />
            Import
          </button>
          <button
            className="btn-primary flex items-center gap-2"
            onClick={() => setShowCreate(!showCreate)}
          >
            <Plus size={16} />
            New Profile
          </button>
        </div>
      </div>

      {/* Create Form */}
      {showCreate && (
        <div className="card space-y-4 animate-fade-in">
          <h3 className="text-sm font-semibold text-gray-300">
            Create New Profile
          </h3>
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="text-xs text-gray-500 mb-1 block">
                Profile Name
              </label>
              <input
                type="text"
                className="input-field"
                placeholder="e.g., Main Channel"
                value={newName}
                onChange={(e) => setNewName(e.target.value)}
              />
            </div>
            <div>
              <label className="text-xs text-gray-500 mb-1 block">
                Channel Name
              </label>
              <input
                type="text"
                className="input-field"
                placeholder="e.g., streamer_name"
                value={newChannel}
                onChange={(e) => setNewChannel(e.target.value)}
              />
            </div>
          </div>
          <div className="flex justify-end gap-2">
            <button
              className="btn-ghost"
              onClick={() => setShowCreate(false)}
            >
              Cancel
            </button>
            <button className="btn-primary" onClick={handleCreate}>
              Create Profile
            </button>
          </div>
        </div>
      )}

      {/* Profiles Grid */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        {profiles.map((profile) => (
          <ProfileCard
            key={profile.id}
            profile={profile}
            onActivate={() => handleActivate(profile.id)}
            onDelete={() => handleDelete(profile.id)}
            onDuplicate={() => handleDuplicate(profile)}
          />
        ))}

        {profiles.length === 0 && (
          <div className="col-span-2 card flex flex-col items-center justify-center py-12 text-gray-600">
            <Users size={48} className="mb-3 opacity-30" />
            <p className="text-lg font-medium">No profiles yet</p>
            <p className="text-sm mt-1">
              Create your first profile to get started
            </p>
          </div>
        )}
      </div>
    </div>
  );
}

interface ProfileCardProps {
  profile: ProfileConfig;
  onActivate: () => void;
  onDelete: () => void;
  onDuplicate: () => void;
}

function ProfileCard({
  profile,
  onActivate,
  onDelete,
  onDuplicate,
}: ProfileCardProps) {
  return (
    <div
      className={`
        card relative transition-all duration-200
        ${
          profile.active
            ? "border-blue-500/40 bg-blue-500/5 ring-1 ring-blue-500/20"
            : "hover:border-gray-700"
        }
      `}
    >
      {/* Active indicator */}
      {profile.active && (
        <div className="absolute top-3 right-3">
          <span className="badge-success flex items-center gap-1">
            <CheckCircle2 size={12} />
            Active
          </span>
        </div>
      )}

      <div className="flex items-start gap-3">
        <div
          className={`
            p-2.5 rounded-xl
            ${profile.active ? "bg-blue-500/20" : "bg-gray-800"}
          `}
        >
          <Tv
            size={20}
            className={profile.active ? "text-blue-400" : "text-gray-500"}
          />
        </div>

        <div className="flex-1 min-w-0">
          <h3 className="text-base font-semibold text-gray-100 truncate">
            {profile.name}
          </h3>
          <p className="text-sm text-gray-500 mt-0.5">
            <span className="badge-info mr-2">{profile.platform}</span>
            #{profile.channel}
          </p>

          {profile.maxWorkers && (
            <p className="text-xs text-gray-600 mt-2">
              {profile.maxWorkers} workers
              {profile.features && (
                <>
                  {" · "}
                  {
                    Object.values(profile.features).filter(Boolean).length
                  }{" "}
                  features enabled
                </>
              )}
            </p>
          )}
        </div>
      </div>

      {/* Actions */}
      <div className="flex items-center gap-1 mt-4 pt-3 border-t border-gray-800/50">
        {!profile.active && (
          <button
            className="btn-ghost text-xs flex items-center gap-1 text-blue-400 hover:text-blue-300"
            onClick={onActivate}
          >
            <Circle size={12} />
            Activate
          </button>
        )}
        <button className="btn-ghost text-xs flex items-center gap-1">
          <Edit3 size={12} />
          Edit
        </button>
        <button
          className="btn-ghost text-xs flex items-center gap-1"
          onClick={onDuplicate}
        >
          <Copy size={12} />
          Duplicate
        </button>
        <div className="flex-1" />
        <button
          className="btn-ghost text-xs flex items-center gap-1 text-red-400 hover:text-red-300"
          onClick={onDelete}
        >
          <Trash2 size={12} />
        </button>
      </div>
    </div>
  );
}
