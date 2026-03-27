import { useEffect, useState, useCallback, createContext, useContext } from "react";
import { X, CheckCircle2, AlertTriangle, AlertCircle, Info, Radio } from "lucide-react";

// Toast types and their visual configs
type ToastType = "success" | "error" | "warning" | "info" | "stream-online" | "stream-offline";

interface Toast {
  id: string;
  type: ToastType;
  title: string;
  message?: string;
  duration?: number; // ms, 0 = persistent
}

const toastConfig: Record<ToastType, { icon: React.ReactNode; colors: string }> = {
  success: {
    icon: <CheckCircle2 size={18} />,
    colors: "bg-emerald-500/10 border-emerald-500/30 text-emerald-300",
  },
  error: {
    icon: <AlertCircle size={18} />,
    colors: "bg-red-500/10 border-red-500/30 text-red-300",
  },
  warning: {
    icon: <AlertTriangle size={18} />,
    colors: "bg-amber-500/10 border-amber-500/30 text-amber-300",
  },
  info: {
    icon: <Info size={18} />,
    colors: "bg-blue-500/10 border-blue-500/30 text-blue-300",
  },
  "stream-online": {
    icon: <Radio size={18} className="animate-pulse" />,
    colors: "bg-red-500/10 border-red-500/30 text-red-300",
  },
  "stream-offline": {
    icon: <Radio size={18} />,
    colors: "bg-gray-500/10 border-gray-500/30 text-gray-300",
  },
};

// Toast context for app-wide notifications
interface ToastContextType {
  addToast: (toast: Omit<Toast, "id">) => void;
  removeToast: (id: string) => void;
}

const ToastContext = createContext<ToastContextType>({
  addToast: () => {},
  removeToast: () => {},
});

export function useToast() {
  return useContext(ToastContext);
}

// Toast provider wraps the app and renders toast container
export function ToastProvider({ children }: { children: React.ReactNode }) {
  const [toasts, setToasts] = useState<Toast[]>([]);

  const addToast = useCallback((toast: Omit<Toast, "id">) => {
    const id = `toast-${Date.now()}-${Math.random().toString(36).slice(2, 6)}`;
    const newToast: Toast = { ...toast, id, duration: toast.duration ?? 5000 };
    setToasts((prev) => [...prev, newToast]);
  }, []);

  const removeToast = useCallback((id: string) => {
    setToasts((prev) => prev.filter((t) => t.id !== id));
  }, []);

  return (
    <ToastContext.Provider value={{ addToast, removeToast }}>
      {children}
      <ToastContainer toasts={toasts} onRemove={removeToast} />
    </ToastContext.Provider>
  );
}

// Toast container renders all active toasts
function ToastContainer({
  toasts,
  onRemove,
}: {
  toasts: Toast[];
  onRemove: (id: string) => void;
}) {
  return (
    <div className="fixed bottom-4 right-4 z-50 flex flex-col gap-2 max-w-sm">
      {toasts.map((toast) => (
        <ToastItem key={toast.id} toast={toast} onRemove={onRemove} />
      ))}
    </div>
  );
}

// Individual toast item with auto-dismiss
function ToastItem({
  toast,
  onRemove,
}: {
  toast: Toast;
  onRemove: (id: string) => void;
}) {
  const config = toastConfig[toast.type];

  useEffect(() => {
    if (toast.duration && toast.duration > 0) {
      const timer = setTimeout(() => onRemove(toast.id), toast.duration);
      return () => clearTimeout(timer);
    }
  }, [toast.id, toast.duration, onRemove]);

  return (
    <div
      className={`
        flex items-start gap-3 px-4 py-3 rounded-xl border backdrop-blur-sm
        animate-slide-up shadow-lg shadow-black/20
        ${config.colors}
      `}
    >
      <span className="mt-0.5 shrink-0">{config.icon}</span>
      <div className="flex-1 min-w-0">
        <p className="text-sm font-semibold">{toast.title}</p>
        {toast.message && (
          <p className="text-xs opacity-70 mt-0.5 truncate">{toast.message}</p>
        )}
      </div>
      <button
        onClick={() => onRemove(toast.id)}
        className="shrink-0 p-0.5 hover:opacity-100 opacity-50 transition-opacity"
      >
        <X size={14} />
      </button>
    </div>
  );
}
