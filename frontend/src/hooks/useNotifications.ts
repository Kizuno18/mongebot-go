// Hook for receiving stream events and displaying toast notifications.
import { useEffect } from "react";
import { ipc } from "../services/ipc";
import { useToast } from "../components/Toast";

interface StreamEvent {
  channel: string;
  platform: string;
  status: number; // 0=offline, 1=online
  metadata?: {
    title?: string;
    game?: string;
    viewerCount?: number;
  };
}

// useStreamNotifications subscribes to stream events and shows toasts.
export function useStreamNotifications() {
  const { addToast } = useToast();

  useEffect(() => {
    const unsub = ipc.on("event.stream", (data) => {
      const event = data as StreamEvent;

      if (event.status === 1) {
        addToast({
          type: "stream-online",
          title: `${event.channel} is LIVE!`,
          message: event.metadata?.title
            ? `${event.metadata.title} — ${event.metadata.game || "No category"}`
            : undefined,
          duration: 10000,
        });
      } else {
        addToast({
          type: "stream-offline",
          title: `${event.channel} went offline`,
          duration: 5000,
        });
      }
    });

    // Engine errors
    const unsubErr = ipc.on("event.error", (data) => {
      const err = data as { message: string; component?: string };
      addToast({
        type: "error",
        title: err.component ? `Error in ${err.component}` : "Engine Error",
        message: err.message,
        duration: 8000,
      });
    });

    return () => {
      unsub();
      unsubErr();
    };
  }, [addToast]);
}
