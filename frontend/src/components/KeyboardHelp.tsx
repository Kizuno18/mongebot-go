import { useEffect } from "react";
import { X, Keyboard } from "lucide-react";

interface KeyboardHelpProps {
  open: boolean;
  onClose: () => void;
}

const shortcuts = [
  { section: "Navigation", items: [
    { keys: ["Ctrl", "1"], desc: "Dashboard" },
    { keys: ["Ctrl", "2"], desc: "Profiles" },
    { keys: ["Ctrl", "3"], desc: "Proxies" },
    { keys: ["Ctrl", "4"], desc: "Tokens" },
    { keys: ["Ctrl", "5"], desc: "Stream Monitor" },
    { keys: ["Ctrl", "6"], desc: "Session History" },
    { keys: ["Ctrl", "7"], desc: "Scheduler" },
    { keys: ["Ctrl", "8"], desc: "Logs" },
    { keys: ["Ctrl", "9"], desc: "Settings" },
  ]},
  { section: "Engine", items: [
    { keys: ["Space"], desc: "Toggle engine start/stop" },
    { keys: ["Escape"], desc: "Stop engine" },
  ]},
  { section: "General", items: [
    { keys: ["?"], desc: "Show this help" },
  ]},
];

export default function KeyboardHelp({ open, onClose }: KeyboardHelpProps) {
  useEffect(() => {
    if (!open) return;
    const handler = (e: KeyboardEvent) => {
      if (e.key === "Escape" || e.key === "?") {
        e.preventDefault();
        onClose();
      }
    };
    window.addEventListener("keydown", handler);
    return () => window.removeEventListener("keydown", handler);
  }, [open, onClose]);

  if (!open) return null;

  return (
    <div className="fixed inset-0 z-[150] flex items-center justify-center">
      <div className="absolute inset-0 bg-gray-950/60 backdrop-blur-sm" onClick={onClose} />

      <div className="relative bg-gray-900 border border-gray-800 rounded-2xl p-6 max-w-md w-full mx-4 shadow-2xl animate-slide-up">
        <div className="flex items-center justify-between mb-5">
          <div className="flex items-center gap-2">
            <Keyboard size={18} className="text-blue-400" />
            <h2 className="text-lg font-bold text-gray-100">
              Keyboard Shortcuts
            </h2>
          </div>
          <button onClick={onClose} className="text-gray-600 hover:text-gray-400">
            <X size={16} />
          </button>
        </div>

        <div className="space-y-5">
          {shortcuts.map((section) => (
            <div key={section.section}>
              <h3 className="text-xs font-semibold text-gray-500 uppercase tracking-wider mb-2">
                {section.section}
              </h3>
              <div className="space-y-1.5">
                {section.items.map((item) => (
                  <div
                    key={item.desc}
                    className="flex items-center justify-between py-1"
                  >
                    <span className="text-sm text-gray-300">{item.desc}</span>
                    <div className="flex items-center gap-1">
                      {item.keys.map((key, i) => (
                        <span key={i}>
                          <kbd className="px-2 py-0.5 bg-gray-800 border border-gray-700 rounded text-[11px] font-mono text-gray-300">
                            {key}
                          </kbd>
                          {i < item.keys.length - 1 && (
                            <span className="text-gray-600 mx-0.5">+</span>
                          )}
                        </span>
                      ))}
                    </div>
                  </div>
                ))}
              </div>
            </div>
          ))}
        </div>

        <p className="text-[10px] text-gray-700 mt-4 text-center">
          Press <kbd className="px-1 py-0.5 bg-gray-800 border border-gray-700 rounded text-[10px]">?</kbd> or <kbd className="px-1 py-0.5 bg-gray-800 border border-gray-700 rounded text-[10px]">Esc</kbd> to close
        </p>
      </div>
    </div>
  );
}
