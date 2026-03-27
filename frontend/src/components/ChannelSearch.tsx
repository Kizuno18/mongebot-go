import { useState, useEffect, useRef } from "react";
import { Search, Radio, Tv } from "lucide-react";
import { ipc } from "../services/ipc";

interface ChannelResult {
  login: string;
  displayName: string;
  id: string;
  isLive: boolean;
  gameName?: string;
  viewerCount: number;
  avatarUrl?: string;
}

interface ChannelSearchProps {
  value: string;
  onChange: (value: string) => void;
  onSelect?: (channel: ChannelResult) => void;
  disabled?: boolean;
  placeholder?: string;
}

export default function ChannelSearch({
  value,
  onChange,
  onSelect,
  disabled,
  placeholder = "Search channel...",
}: ChannelSearchProps) {
  const [results, setResults] = useState<ChannelResult[]>([]);
  const [open, setOpen] = useState(false);
  const [loading, setLoading] = useState(false);
  const debounceRef = useRef<ReturnType<typeof setTimeout>>();
  const containerRef = useRef<HTMLDivElement>(null);

  // Debounced search
  useEffect(() => {
    if (debounceRef.current) clearTimeout(debounceRef.current);

    if (value.length < 2) {
      setResults([]);
      setOpen(false);
      return;
    }

    debounceRef.current = setTimeout(async () => {
      setLoading(true);
      try {
        const data = await ipc.call<ChannelResult[]>("channel.search", {
          query: value,
          limit: 8,
        });
        setResults(data || []);
        setOpen(true);
      } catch {
        setResults([]);
      } finally {
        setLoading(false);
      }
    }, 300);

    return () => {
      if (debounceRef.current) clearTimeout(debounceRef.current);
    };
  }, [value]);

  // Close dropdown on outside click
  useEffect(() => {
    const handler = (e: MouseEvent) => {
      if (containerRef.current && !containerRef.current.contains(e.target as Node)) {
        setOpen(false);
      }
    };
    document.addEventListener("mousedown", handler);
    return () => document.removeEventListener("mousedown", handler);
  }, []);

  const handleSelect = (ch: ChannelResult) => {
    onChange(ch.login);
    setOpen(false);
    onSelect?.(ch);
  };

  return (
    <div ref={containerRef} className="relative flex-1">
      <div className="relative">
        <Search
          size={16}
          className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-500"
        />
        <input
          type="text"
          className="input-field pl-10 pr-8"
          placeholder={placeholder}
          value={value}
          onChange={(e) => onChange(e.target.value)}
          onFocus={() => results.length > 0 && setOpen(true)}
          disabled={disabled}
        />
        {loading && (
          <div className="absolute right-3 top-1/2 -translate-y-1/2">
            <div className="w-4 h-4 border-2 border-gray-600 border-t-blue-400 rounded-full animate-spin" />
          </div>
        )}
      </div>

      {/* Dropdown */}
      {open && results.length > 0 && (
        <div className="absolute z-50 w-full mt-1 bg-gray-900 border border-gray-700 rounded-xl shadow-xl overflow-hidden animate-fade-in">
          {results.map((ch) => (
            <button
              key={ch.id}
              className="w-full flex items-center gap-3 px-3 py-2.5 hover:bg-gray-800 transition-colors text-left"
              onClick={() => handleSelect(ch)}
            >
              <div className="w-8 h-8 rounded-full bg-gray-800 flex items-center justify-center shrink-0">
                <Tv size={14} className="text-gray-500" />
              </div>
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-2">
                  <span className="text-sm font-medium text-gray-200 truncate">
                    {ch.displayName}
                  </span>
                  {ch.isLive && (
                    <span className="flex items-center gap-1 text-[10px] text-red-400 font-semibold">
                      <Radio size={10} className="animate-pulse" />
                      LIVE
                    </span>
                  )}
                </div>
                <div className="flex items-center gap-2 text-xs text-gray-500">
                  <span>{ch.login}</span>
                  {ch.isLive && ch.gameName && (
                    <>
                      <span>·</span>
                      <span className="truncate">{ch.gameName}</span>
                    </>
                  )}
                  {ch.isLive && (
                    <>
                      <span>·</span>
                      <span>{ch.viewerCount.toLocaleString()} viewers</span>
                    </>
                  )}
                </div>
              </div>
            </button>
          ))}
        </div>
      )}
    </div>
  );
}
