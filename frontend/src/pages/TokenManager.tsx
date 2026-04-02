import { useState, useEffect } from "react";
import {
  Upload,
  ShieldCheck,
  ShieldX,
  RefreshCw,
} from "lucide-react";
import { ipc } from "../services/ipc";

interface TokenItem {
  value: string;
  label: string;
  state: string;
  useCount: number;
  errorCount: number;
  platform: string;
}

export default function TokenManager() {
  const [importText, setImportText] = useState("");
  const [showImport, setShowImport] = useState(false);
  const [importMode, setImportMode] = useState<"raw">("raw");
  const [tokens, setTokens] = useState<TokenItem[]>([]);
  const [stats, setStats] = useState<any>({ total: 0, valid: 0, expired: 0, quarantined: 0, inUse: 0 });
  const [loading, setLoading] = useState(false);
  const [validating, setValidating] = useState(false);

  const fetchTokens = async () => {
    setLoading(true);
    try {
      const list = await ipc.call<TokenItem[]>("token.list");
      const currentStats = await ipc.call<any>("token.stats");
      setTokens(list || []);
      setStats(currentStats || { total: 0, valid: 0, expired: 0, quarantined: 0, inUse: 0 });
    } catch (err) {
      console.error("Failed to fetch tokens:", err);
    }
    setLoading(false);
  };

  useEffect(() => {
    fetchTokens();
    const interval = setInterval(fetchTokens, 5000); // refresh every 5s
    return () => clearInterval(interval);
  }, []);

  const handleValidateAll = async () => {
    setValidating(true);
    try {
      await ipc.call("token.validate");
    } catch (err) {
      console.error("Validation failed", err);
    }
    // Don't wait for completion since backend handles it async
    setTimeout(() => {
      setValidating(false);
      fetchTokens();
    }, 2000);
  };

  const handleImport = async () => {
    if (!importText.trim()) return;
    try {
      // Split by newline and filter empty lines
      const rawTokens = importText.split('\n').map(t => t.trim()).filter(t => t.length > 0);
      await ipc.call("token.import", { tokens: rawTokens, platform: "twitch" });
      setShowImport(false);
      setImportText("");
      fetchTokens();
    } catch(err) {
      console.error("Import failed:", err);
    }
  };

  return (
    <div className="h-full overflow-y-auto p-6 space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-100">Token Manager</h1>
          <p className="text-sm text-gray-500 mt-1">
            Manage and Validate your tokens
          </p>
        </div>
        <div className="flex items-center gap-2">
          <button className="btn-ghost flex items-center gap-2" onClick={fetchTokens} disabled={loading}>
            <RefreshCw size={16} className={loading ? "animate-spin" : ""} />
            Refresh
          </button>
          <button 
            className="btn-ghost flex items-center gap-2" 
            onClick={handleValidateAll}
            disabled={validating}
          >
            <ShieldCheck size={16} className={validating ? "animate-pulse text-blue-400" : ""} />
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
      <div className="grid grid-cols-4 gap-4">
        <div className="card text-center">
          <p className="text-3xl font-bold text-blue-400">{stats.total}</p>
          <p className="text-xs text-gray-500 mt-1">Total</p>
        </div>
        <div className="card text-center">
          <p className="text-3xl font-bold text-emerald-400">{stats.valid}</p>
          <p className="text-xs text-gray-500 mt-1">Valid</p>
        </div>
        <div className="card text-center">
          <p className="text-3xl font-bold text-red-400">{stats.expired}</p>
          <p className="text-xs text-gray-500 mt-1">Expired</p>
        </div>
        <div className="card text-center">
          <p className="text-3xl font-bold text-yellow-400">{stats.quarantined}</p>
          <p className="text-xs text-gray-500 mt-1">Quarantined</p>
        </div>
      </div>

      {/* Import Panel */}
      {showImport && (
        <div className="card space-y-4">
          <h3 className="text-sm font-semibold text-gray-300">Import Tokens</h3>
          <div className="flex gap-1 bg-gray-900 rounded-lg p-1">
            {(["raw"] as const).map((mode) => (
              <button
                key={mode}
                className={`flex-1 px-3 py-1.5 rounded-md text-xs font-medium transition-colors ${
                  importMode === mode ? "bg-gray-700 text-gray-100" : "text-gray-500 hover:text-gray-300"
                }`}
                onClick={() => setImportMode(mode as any)}
              >
                Raw Tokens
              </button>
            ))}
          </div>

          <p className="text-xs text-gray-500">
            Paste OAuth tokens, one per line.
          </p>

          <textarea
            className="input-field h-32 font-mono text-sm resize-none"
            placeholder="glf7xrso4zbrn15cl9t26xpbudrpb2&#10;gpdgew57y2p89vale53a0z3gevoji2"
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
            <button className="btn-primary" disabled={!importText.trim()} onClick={handleImport}>
              Import
            </button>
          </div>
        </div>
      )}

      {/* Token List */}
      <div className="space-y-2">
        {tokens.length === 0 && !loading && (
            <div className="text-center text-gray-500 py-10">No tokens found. Import some tokens to begin.</div>
        )}
        {tokens.map((token, i) => (
          <div
            key={i}
            className="card flex items-center justify-between py-3"
          >
            <div className="flex items-center gap-3">
              {token.state === "valid" ? (
                <ShieldCheck size={18} className="text-emerald-400" />
              ) : token.state === "quarantined" ? (
                <ShieldX size={18} className="text-yellow-400" />
              ) : (
                <ShieldX size={18} className="text-red-400" />
              )}
              <div>
                <div className="flex gap-2 items-center">
                    <p className="text-sm font-medium text-gray-200">
                    {token.label || "Unnamed Token"}
                    </p>
                    <span className="text-xs text-gray-500">Uses: {token.useCount}</span>
                    {token.errorCount > 0 && <span className="text-xs text-red-500">Errors: {token.errorCount}</span>}
                </div>
                <p className="text-xs font-mono text-gray-500">
                  {token.value}
                </p>
              </div>
            </div>
            <div className="flex items-center gap-3">
              <span
                className={token.state === "valid" ? "badge-success text-emerald-400" : token.state === "quarantined" ? "badge-warning text-yellow-400" : "badge-danger text-red-400"}
              >
                {token.state}
              </span>
              <span className="badge-info">{token.platform}</span>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
