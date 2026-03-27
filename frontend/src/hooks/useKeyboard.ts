// Global keyboard shortcuts for MongeBot desktop app.
import { useEffect } from "react";
import { ipc } from "../services/ipc";
import type { EngineMetrics } from "../types";

interface KeyboardOptions {
  onToggleEngine?: () => void;
  onNavigate?: (page: string) => void;
  metrics?: EngineMetrics | null;
}

// useKeyboard registers global keyboard shortcuts.
export function useKeyboard({ onToggleEngine, onNavigate, metrics }: KeyboardOptions) {
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      // Ignore when typing in input fields
      const target = e.target as HTMLElement;
      if (target.tagName === "INPUT" || target.tagName === "TEXTAREA" || target.tagName === "SELECT") {
        return;
      }

      // Ctrl/Cmd + key combinations
      const mod = e.ctrlKey || e.metaKey;

      if (mod) {
        switch (e.key.toLowerCase()) {
          case "1":
            e.preventDefault();
            onNavigate?.("dashboard");
            break;
          case "2":
            e.preventDefault();
            onNavigate?.("profiles");
            break;
          case "3":
            e.preventDefault();
            onNavigate?.("proxies");
            break;
          case "4":
            e.preventDefault();
            onNavigate?.("tokens");
            break;
          case "5":
            e.preventDefault();
            onNavigate?.("stream");
            break;
          case "6":
            e.preventDefault();
            onNavigate?.("history");
            break;
          case "7":
            e.preventDefault();
            onNavigate?.("logs");
            break;
          case "8":
            e.preventDefault();
            onNavigate?.("settings");
            break;
        }
      }

      // Escape — stop engine
      if (e.key === "Escape" && metrics?.engineState === "running") {
        onToggleEngine?.();
      }

      // Space — toggle engine (only on dashboard with no focus)
      if (e.key === " " && !mod) {
        e.preventDefault();
        onToggleEngine?.();
      }
    };

    window.addEventListener("keydown", handler);
    return () => window.removeEventListener("keydown", handler);
  }, [onToggleEngine, onNavigate, metrics]);
}

// useWindowTitle updates the Tauri window title based on engine state.
export function useWindowTitle(metrics: EngineMetrics | null) {
  useEffect(() => {
    let title = "MongeBot";

    if (metrics?.engineState === "running" && metrics.channel) {
      title = `MongeBot — ${metrics.channel} (${metrics.activeViewers}/${metrics.totalWorkers} viewers)`;
    } else if (metrics?.engineState === "starting") {
      title = "MongeBot — Starting...";
    }

    document.title = title;
  }, [metrics?.engineState, metrics?.channel, metrics?.activeViewers, metrics?.totalWorkers]);
}
