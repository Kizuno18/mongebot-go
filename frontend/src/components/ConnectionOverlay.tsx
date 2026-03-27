import { WifiOff, Loader2 } from "lucide-react";

interface ConnectionOverlayProps {
  connected: boolean;
  reconnecting?: boolean;
}

// ConnectionOverlay shows a full-screen overlay when disconnected from the backend.
export default function ConnectionOverlay({
  connected,
  reconnecting,
}: ConnectionOverlayProps) {
  if (connected) return null;

  return (
    <div className="fixed inset-0 z-[100] bg-gray-950/80 backdrop-blur-sm flex items-center justify-center">
      <div className="text-center space-y-4">
        <div className="w-20 h-20 rounded-2xl bg-gray-900 border border-gray-800 flex items-center justify-center mx-auto">
          {reconnecting ? (
            <Loader2 size={32} className="text-blue-400 animate-spin" />
          ) : (
            <WifiOff size={32} className="text-red-400" />
          )}
        </div>
        <div>
          <h2 className="text-xl font-bold text-gray-200">
            {reconnecting ? "Reconnecting..." : "Disconnected"}
          </h2>
          <p className="text-sm text-gray-500 mt-1 max-w-xs">
            {reconnecting
              ? "Attempting to reconnect to the MongeBot backend..."
              : "Lost connection to the backend. Check if MongeBot is running."}
          </p>
        </div>
        {!reconnecting && (
          <button
            className="btn-primary"
            onClick={() => window.location.reload()}
          >
            Reload
          </button>
        )}
      </div>
    </div>
  );
}
