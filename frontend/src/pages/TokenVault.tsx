import { useState } from "react";
import {
  KeyRound,
  Upload,
  ShieldCheck,
  ShieldX,
  ShieldAlert,
  Lock,
} from "lucide-react";

export default function TokenVault() {
  const [importText, setImportText] = useState("");
  const [showImport, setShowImport] = useState(false);
  const [importMode, setImportMode] = useState<"raw" | "cookies" | "json">("raw");

  // Placeholder data (will connect to IPC)
  const tokens = [
    { id: "1", platform: "twitch", value: "abc123...xyz", label: "Account 1", valid: true },
    { id: "2", platform: "twitch", value: "def456...uvw", label: "Account 2", valid: true },
    { id: "3", platform: "twitch", value: "ghi789...rst", label: "Account 3", valid: false },
  ];

  const validCount = tokens.filter((t) => t.valid).length;
  const invalidCount = tokens.filter((t) => !t.valid).length;

  return (
    <div className="h-full overflow-y-auto p-6 space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-100">Token Vault</h1>
          <p className="text-sm text-gray-500 mt-1">
            Encrypted storage for authentication tokens
          </p>
        </div>
        <div className="flex items-center gap-2">
          <button className="btn-ghost flex items-center gap-2">
            <ShieldCheck size={16} />
            Validate All
          </button>
          <button
            className="btn-primary flex items-center gap-2"
            onClick={() => setShowImport(!showImport)}
          >
            <Upload size={16} />
            Import
          </button>
        </div>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-3 gap-4">
        <div className="card text-center">
          <p className="text-3xl font-bold text-blue-400">{tokens.length}</p>
          <p className="text-xs text-gray-500 mt-1">Total Tokens</p>
        </div>
        <div className="card text-center">
          <p className="text-3xl font-bold text-emerald-400">{validCount}</p>
          <p className="text-xs text-gray-500 mt-1">Valid</p>
        </div>
        <div className="card text-center">
          <p className="text-3xl font-bold text-red-400">{invalidCount}</p>
          <p className="text-xs text-gray-500 mt-1">Invalid</p>
        </div>
      </div>

      {/* Encryption notice */}
      <div className="card flex items-center gap-3 bg-blue-500/5 border-blue-500/20">
        <Lock size={20} className="text-blue-400 shrink-0" />
        <div>
          <p className="text-sm text-blue-300">AES-256-GCM Encrypted</p>
          <p className="text-xs text-gray-500">
            All tokens are encrypted at rest using PBKDF2-derived keys
          </p>
        </div>
      </div>

      {/* Import Panel */}
      {showImport && (
        <div className="card space-y-4">
          <h3 className="text-sm font-semibold text-gray-300">Import Tokens</h3>

          {/* Import mode tabs */}
          <div className="flex gap-1 bg-gray-900 rounded-lg p-1">
            {(["raw", "cookies", "json"] as const).map((mode) => (
              <button
                key={mode}
                className={`flex-1 px-3 py-1.5 rounded-md text-xs font-medium transition-colors ${
                  importMode === mode ? "bg-gray-700 text-gray-100" : "text-gray-500 hover:text-gray-300"
                }`}
                onClick={() => setImportMode(mode)}
              >
                {mode === "raw" ? "Raw Tokens" : mode === "cookies" ? "Browser Cookies" : "JSON Array"}
              </button>
            ))}
          </div>

          <p className="text-xs text-gray-500">
            {importMode === "raw" && "Paste OAuth tokens, one per line."}
            {importMode === "cookies" && "Paste Netscape cookie export or EditThisCookie JSON. auth-token will be extracted automatically."}
            {importMode === "json" && "Paste a JSON array of token strings."}
          </p>

          <textarea
            className="input-field h-32 font-mono text-sm resize-none"
            placeholder={
              importMode === "raw" ? "abc123def456...\nghi789jkl012..." :
              importMode === "cookies" ? '.twitch.tv\tTRUE\t/\tTRUE\t0\tauth-token\tyour_token_here' :
              '["token1", "token2", "token3"]'
            }
            value={importText}
            onChange={(e) => setImportText(e.target.value)}
          />

          {importText && (
            <div className="flex items-center gap-2 text-xs text-gray-500">
              <span className="w-2 h-2 rounded-full bg-blue-500" />
              {importText.split("\n").filter((l) => l.trim()).length} lines detected
            </div>
          )}

          <div className="flex justify-end gap-2">
            <button className="btn-ghost" onClick={() => { setShowImport(false); setImportText(""); }}>
              Cancel
            </button>
            <button className="btn-primary" disabled={!importText.trim()}>
              Encrypt & Import
            </button>
          </div>
        </div>
      )}

      {/* Token List */}
      <div className="space-y-2">
        {tokens.map((token) => (
          <div
            key={token.id}
            className="card flex items-center justify-between py-3"
          >
            <div className="flex items-center gap-3">
              {token.valid ? (
                <ShieldCheck size={18} className="text-emerald-400" />
              ) : (
                <ShieldX size={18} className="text-red-400" />
              )}
              <div>
                <p className="text-sm font-medium text-gray-200">
                  {token.label || "Unnamed Token"}
                </p>
                <p className="text-xs font-mono text-gray-500">
                  {token.value.slice(0, 8)}...{token.value.slice(-4)}
                </p>
              </div>
            </div>
            <div className="flex items-center gap-3">
              <span
                className={token.valid ? "badge-success" : "badge-danger"}
              >
                {token.valid ? "Valid" : "Expired"}
              </span>
              <span className="badge-info">{token.platform}</span>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
