import { useState } from "react";
import {
  Zap,
  ArrowRight,
  ArrowLeft,
  Check,
  Network,
  KeyRound,
  Users,
  Settings,
  Sparkles,
} from "lucide-react";

interface OnboardingWizardProps {
  onComplete: () => void;
}

const steps = [
  { id: "welcome", title: "Welcome", icon: <Zap size={20} /> },
  { id: "proxies", title: "Proxies", icon: <Network size={20} /> },
  { id: "tokens", title: "Tokens", icon: <KeyRound size={20} /> },
  { id: "profile", title: "Profile", icon: <Users size={20} /> },
  { id: "done", title: "Ready!", icon: <Sparkles size={20} /> },
];

export default function OnboardingWizard({ onComplete }: OnboardingWizardProps) {
  const [step, setStep] = useState(0);
  const [proxyText, setProxyText] = useState("");
  const [tokenText, setTokenText] = useState("");
  const [profileName, setProfileName] = useState("");
  const [channelName, setChannelName] = useState("");

  const canNext = () => {
    switch (step) {
      case 0:
        return true; // Welcome is always valid
      case 1:
        return true; // Proxies are optional
      case 2:
        return true; // Tokens are optional (can add later)
      case 3:
        return channelName.trim().length > 0;
      default:
        return true;
    }
  };

  const handleFinish = () => {
    // Save the setup flag to localStorage
    localStorage.setItem("mongebot-setup-complete", "true");
    onComplete();
  };

  return (
    <div className="fixed inset-0 z-[200] bg-gray-950 flex items-center justify-center">
      <div className="w-full max-w-xl mx-auto p-8 space-y-8">
        {/* Progress bar */}
        <div className="flex items-center gap-1">
          {steps.map((s, i) => (
            <div
              key={s.id}
              className={`flex-1 h-1 rounded-full transition-colors ${
                i <= step ? "bg-blue-500" : "bg-gray-800"
              }`}
            />
          ))}
        </div>

        {/* Step content */}
        <div className="min-h-[320px] flex flex-col">
          {step === 0 && (
            <div className="flex-1 flex flex-col items-center justify-center text-center space-y-4">
              <div className="w-20 h-20 rounded-2xl bg-blue-600/20 flex items-center justify-center">
                <Zap size={40} className="text-blue-400" />
              </div>
              <h1 className="text-3xl font-bold text-gray-100">
                Welcome to MongeBot
              </h1>
              <p className="text-gray-500 max-w-md">
                Let's set up your viewer bot in a few quick steps. You can always
                change these settings later.
              </p>
            </div>
          )}

          {step === 1 && (
            <div className="flex-1 space-y-4">
              <div className="flex items-center gap-3">
                <Network size={24} className="text-blue-400" />
                <div>
                  <h2 className="text-xl font-bold text-gray-100">
                    Import Proxies
                  </h2>
                  <p className="text-sm text-gray-500">
                    Paste your proxy list (optional — you can add later)
                  </p>
                </div>
              </div>
              <textarea
                className="input-field h-40 font-mono text-sm resize-none"
                placeholder="ip:port:user:pass (one per line)..."
                value={proxyText}
                onChange={(e) => setProxyText(e.target.value)}
              />
              {proxyText && (
                <p className="text-xs text-gray-500">
                  {proxyText.split("\n").filter((l) => l.trim()).length} proxies
                  detected
                </p>
              )}
            </div>
          )}

          {step === 2 && (
            <div className="flex-1 space-y-4">
              <div className="flex items-center gap-3">
                <KeyRound size={24} className="text-emerald-400" />
                <div>
                  <h2 className="text-xl font-bold text-gray-100">
                    Import Tokens
                  </h2>
                  <p className="text-sm text-gray-500">
                    Paste OAuth tokens or browser cookies (optional)
                  </p>
                </div>
              </div>
              <textarea
                className="input-field h-40 font-mono text-sm resize-none"
                placeholder="Paste tokens here..."
                value={tokenText}
                onChange={(e) => setTokenText(e.target.value)}
              />
              {tokenText && (
                <p className="text-xs text-gray-500">
                  {tokenText.split("\n").filter((l) => l.trim()).length} lines
                  detected
                </p>
              )}
            </div>
          )}

          {step === 3 && (
            <div className="flex-1 space-y-4">
              <div className="flex items-center gap-3">
                <Users size={24} className="text-amber-400" />
                <div>
                  <h2 className="text-xl font-bold text-gray-100">
                    Create First Profile
                  </h2>
                  <p className="text-sm text-gray-500">
                    Set up your first target channel
                  </p>
                </div>
              </div>
              <div className="space-y-3">
                <div>
                  <label className="text-xs text-gray-500 mb-1 block">
                    Profile Name
                  </label>
                  <input
                    type="text"
                    className="input-field"
                    placeholder="e.g., Main Channel"
                    value={profileName}
                    onChange={(e) => setProfileName(e.target.value)}
                  />
                </div>
                <div>
                  <label className="text-xs text-gray-500 mb-1 block">
                    Channel Name *
                  </label>
                  <input
                    type="text"
                    className="input-field"
                    placeholder="e.g., streamer_name"
                    value={channelName}
                    onChange={(e) => setChannelName(e.target.value)}
                  />
                </div>
              </div>
            </div>
          )}

          {step === 4 && (
            <div className="flex-1 flex flex-col items-center justify-center text-center space-y-4">
              <div className="w-20 h-20 rounded-2xl bg-emerald-600/20 flex items-center justify-center">
                <Sparkles size={40} className="text-emerald-400" />
              </div>
              <h1 className="text-3xl font-bold text-gray-100">
                You're All Set!
              </h1>
              <p className="text-gray-500 max-w-md">
                MongeBot is ready to use. Head to the Dashboard to start your
                first session.
              </p>
              <div className="flex flex-col gap-1 text-sm text-gray-600 mt-2">
                {proxyText && (
                  <span>
                    {proxyText.split("\n").filter((l) => l.trim()).length}{" "}
                    proxies imported
                  </span>
                )}
                {tokenText && (
                  <span>
                    {tokenText.split("\n").filter((l) => l.trim()).length}{" "}
                    tokens imported
                  </span>
                )}
                {channelName && <span>Profile: #{channelName}</span>}
              </div>
            </div>
          )}
        </div>

        {/* Navigation */}
        <div className="flex items-center justify-between">
          <button
            className="btn-ghost flex items-center gap-2"
            onClick={() => setStep(Math.max(0, step - 1))}
            disabled={step === 0}
          >
            <ArrowLeft size={16} />
            Back
          </button>

          <div className="flex items-center gap-2">
            {step < 4 && (
              <button
                className="btn-ghost text-sm text-gray-600"
                onClick={handleFinish}
              >
                Skip Setup
              </button>
            )}

            {step < 4 ? (
              <button
                className="btn-primary flex items-center gap-2"
                onClick={() => setStep(step + 1)}
                disabled={!canNext()}
              >
                Next
                <ArrowRight size={16} />
              </button>
            ) : (
              <button
                className="btn-primary flex items-center gap-2"
                onClick={handleFinish}
              >
                <Check size={16} />
                Start Using MongeBot
              </button>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
