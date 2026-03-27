import { useState, useEffect, useCallback } from "react";
import {
  Upload,
  RefreshCw,
  Trash2,
  Globe,
  Wifi,
  WifiOff,
  AlertTriangle,
} from "lucide-react";
import { ipc } from "../services/ipc";
import type { ProxyInfo, ProxyListResponse } from "../types";

const healthIcons: Record<number, { icon: React.ReactNode; label: string }> = {
  0: { icon: <AlertTriangle size={14} className="text-gray-400" />, label: "Unknown" },
  1: { icon: <Wifi size={14} className="text-emerald-400" />, label: "Good" },
  2: { icon: <Wifi size={14} className="text-amber-400" />, label: "Slow" },
  3: { icon: <WifiOff size={14} className="text-red-400" />, label: "Dead" },
};

export default function ProxyManager() {
  const [proxies, setProxies] = useState<ProxyInfo[]>([]);
  const [stats, setStats] = useState({ total: 0, available: 0, inUse: 0 });
  const [importText, setImportText] = useState("");
  const [showImport, setShowImport] = useState(false);

  const loadProxies = useCallback(async () => {
    try {
      const data = await ipc.call<ProxyListResponse>("proxy.list");
      setProxies(data.proxies || []);
      setStats({ total: data.total, available: data.available, inUse: data.inUse });
    } catch {
      // Not connected yet
    }
  }, []);

  useEffect(() => {
    loadProxies();
  }, [loadProxies]);

  const handleImport = async () => {
    const lines = importText.split("\n").filter((l) => l.trim());
    if (lines.length === 0) return;

    try {
      const result = await ipc.call<{ added: number; errors: string[] }>(
        "proxy.import",
        { proxies: lines },
      );
      setImportText("");
      setShowImport(false);
      loadProxies();
      alert(`Added ${result.added} proxies. ${result.errors?.length || 0} errors.`);
    } catch (err) {
      alert(`Import failed: ${err}`);
    }
  };

  const [scraping, setScraping] = useState(false);
  const [scrapeResult, setScrapeResult] = useState<number | null>(null);

  const handleCheckAll = async () => {
    try {
      await ipc.call("proxy.check");
    } catch {
      // Ignore
    }
  };

  const handleScrape = async () => {
    setScraping(true);
    setScrapeResult(null);
    try {
      const result = await ipc.call<{ found: number }>("proxy.scrape");
      setScrapeResult(result.found);
      loadProxies(); // Refresh list
    } catch {
      setScrapeResult(-1);
    } finally {
      setScraping(false);
    }
  };

  return (
    <div className="h-full overflow-y-auto p-6 space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-100">Proxy Manager</h1>
          <p className="text-sm text-gray-500 mt-1">
            Manage and monitor your proxy pool
          </p>
        </div>
        <div className="flex items-center gap-2">
          <button
            className="btn-ghost flex items-center gap-2"
            onClick={handleScrape}
            disabled={scraping}
          >
            <Globe size={16} className={scraping ? "animate-spin" : ""} />
            {scraping ? "Scraping..." : "Auto-Scrape"}
          </button>
          <button className="btn-ghost flex items-center gap-2" onClick={handleCheckAll}>
            <RefreshCw size={16} />
            Check All
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
          <p className="text-3xl font-bold text-blue-400">{stats.total}</p>
          <p className="text-xs text-gray-500 mt-1">Total</p>
        </div>
        <div className="card text-center">
          <p className="text-3xl font-bold text-emerald-400">{stats.available}</p>
          <p className="text-xs text-gray-500 mt-1">Available</p>
        </div>
        <div className="card text-center">
          <p className="text-3xl font-bold text-amber-400">{stats.inUse}</p>
          <p className="text-xs text-gray-500 mt-1">In Use</p>
        </div>
      </div>

      {/* Scrape Result */}
      {scrapeResult !== null && (
        <div
          className={`card flex items-center gap-3 animate-fade-in ${
            scrapeResult >= 0
              ? "bg-emerald-500/5 border-emerald-500/20"
              : "bg-red-500/5 border-red-500/20"
          }`}
        >
          <Globe size={18} className={scrapeResult >= 0 ? "text-emerald-400" : "text-red-400"} />
          <p className="text-sm">
            {scrapeResult >= 0
              ? `Found ${scrapeResult} proxies from public APIs. Added to pool.`
              : "Scraping failed. Check your network connection."}
          </p>
        </div>
      )}

      {/* Import Panel */}
      {showImport && (
        <div className="card space-y-3">
          <h3 className="text-sm font-semibold text-gray-300">
            Import Proxies
          </h3>
          <p className="text-xs text-gray-500">
            Supported formats: ip:port, ip:port:user:pass, socks5://user:pass@ip:port
          </p>
          <textarea
            className="input-field h-32 font-mono text-sm resize-none"
            placeholder="Paste proxies here, one per line..."
            value={importText}
            onChange={(e) => setImportText(e.target.value)}
          />
          <div className="flex justify-end gap-2">
            <button className="btn-ghost" onClick={() => setShowImport(false)}>
              Cancel
            </button>
            <button className="btn-primary" onClick={handleImport}>
              Import ({importText.split("\n").filter((l) => l.trim()).length})
            </button>
          </div>
        </div>
      )}

      {/* Proxy List */}
      <div className="card overflow-hidden p-0">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-gray-800">
              <th className="text-left p-3 text-gray-500 font-medium">Status</th>
              <th className="text-left p-3 text-gray-500 font-medium">Host</th>
              <th className="text-left p-3 text-gray-500 font-medium">Port</th>
              <th className="text-left p-3 text-gray-500 font-medium">Type</th>
              <th className="text-left p-3 text-gray-500 font-medium">Latency</th>
              <th className="text-left p-3 text-gray-500 font-medium">Uses</th>
              <th className="text-right p-3 text-gray-500 font-medium">Actions</th>
            </tr>
          </thead>
          <tbody>
            {proxies.length === 0 ? (
              <tr>
                <td colSpan={7} className="p-8 text-center text-gray-600">
                  <Globe size={32} className="mx-auto mb-2 opacity-30" />
                  No proxies loaded. Click "Import" to add proxies.
                </td>
              </tr>
            ) : (
              proxies.map((proxy, i) => (
                <tr
                  key={i}
                  className="border-b border-gray-800/50 hover:bg-gray-800/30 transition-colors"
                >
                  <td className="p-3">
                    <div className="flex items-center gap-1.5">
                      {healthIcons[proxy.health]?.icon}
                      <span className="text-xs text-gray-500">
                        {healthIcons[proxy.health]?.label}
                      </span>
                    </div>
                  </td>
                  <td className="p-3 font-mono text-gray-300">{proxy.host}</td>
                  <td className="p-3 font-mono text-gray-400">{proxy.port}</td>
                  <td className="p-3">
                    <span className="badge-info">
                      {["HTTP", "SOCKS4", "SOCKS5"][proxy.type]}
                    </span>
                  </td>
                  <td className="p-3 font-mono text-gray-400">
                    {proxy.latency > 0 ? `${Math.round(proxy.latency / 1e6)}ms` : "—"}
                  </td>
                  <td className="p-3 text-gray-400">{proxy.useCount}</td>
                  <td className="p-3 text-right">
                    <button className="p-1.5 hover:bg-gray-700 rounded transition-colors text-gray-500 hover:text-red-400">
                      <Trash2 size={14} />
                    </button>
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}
