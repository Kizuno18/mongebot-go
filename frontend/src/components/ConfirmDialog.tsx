import { useEffect, useRef } from "react";
import { AlertTriangle, X } from "lucide-react";

interface ConfirmDialogProps {
  open: boolean;
  title: string;
  message: string;
  confirmLabel?: string;
  cancelLabel?: string;
  variant?: "danger" | "warning" | "default";
  onConfirm: () => void;
  onCancel: () => void;
}

export default function ConfirmDialog({
  open,
  title,
  message,
  confirmLabel = "Confirm",
  cancelLabel = "Cancel",
  variant = "danger",
  onConfirm,
  onCancel,
}: ConfirmDialogProps) {
  const confirmRef = useRef<HTMLButtonElement>(null);

  // Focus confirm button when dialog opens
  useEffect(() => {
    if (open) {
      confirmRef.current?.focus();
    }
  }, [open]);

  // Close on Escape
  useEffect(() => {
    if (!open) return;
    const handler = (e: KeyboardEvent) => {
      if (e.key === "Escape") onCancel();
    };
    window.addEventListener("keydown", handler);
    return () => window.removeEventListener("keydown", handler);
  }, [open, onCancel]);

  if (!open) return null;

  const confirmColors = {
    danger: "bg-red-600 hover:bg-red-500",
    warning: "bg-amber-600 hover:bg-amber-500",
    default: "bg-blue-600 hover:bg-blue-500",
  };

  return (
    <div className="fixed inset-0 z-[150] flex items-center justify-center">
      {/* Backdrop */}
      <div
        className="absolute inset-0 bg-gray-950/60 backdrop-blur-sm"
        onClick={onCancel}
      />

      {/* Dialog */}
      <div className="relative bg-gray-900 border border-gray-800 rounded-2xl p-6 max-w-sm w-full mx-4 shadow-2xl animate-slide-up">
        <button
          onClick={onCancel}
          className="absolute top-4 right-4 text-gray-600 hover:text-gray-400 transition-colors"
        >
          <X size={16} />
        </button>

        <div className="flex items-start gap-4">
          <div
            className={`p-2.5 rounded-xl shrink-0 ${
              variant === "danger"
                ? "bg-red-500/10"
                : variant === "warning"
                  ? "bg-amber-500/10"
                  : "bg-blue-500/10"
            }`}
          >
            <AlertTriangle
              size={20}
              className={
                variant === "danger"
                  ? "text-red-400"
                  : variant === "warning"
                    ? "text-amber-400"
                    : "text-blue-400"
              }
            />
          </div>
          <div>
            <h3 className="text-base font-semibold text-gray-100">{title}</h3>
            <p className="text-sm text-gray-400 mt-1">{message}</p>
          </div>
        </div>

        <div className="flex justify-end gap-2 mt-6">
          <button className="btn-ghost" onClick={onCancel}>
            {cancelLabel}
          </button>
          <button
            ref={confirmRef}
            className={`px-4 py-2 text-white rounded-lg font-medium transition-colors ${confirmColors[variant]}`}
            onClick={onConfirm}
          >
            {confirmLabel}
          </button>
        </div>
      </div>
    </div>
  );
}
