import { useState, useEffect } from "react";
import {
  LayoutDashboard,
  Network,
  KeyRound,
  Settings,
  ScrollText,
  Radio,
  Users,
  Zap,
  MonitorPlay,
  History,
  CalendarClock,
} from "lucide-react";
import { useConnection, useMetrics, useEngineControl } from "./hooks/useIPC";
import { useStreamNotifications } from "./hooks/useNotifications";
import { useKeyboard, useWindowTitle } from "./hooks/useKeyboard";
import StatusBar from "./components/StatusBar";
import ConnectionOverlay from "./components/ConnectionOverlay";
import OnboardingWizard from "./components/OnboardingWizard";
import KeyboardHelp from "./components/KeyboardHelp";
import Dashboard from "./pages/Dashboard";
import ProxyManager from "./pages/ProxyManager";
import TokenVault from "./pages/TokenVault";
import ProfilesPage from "./pages/ProfilesPage";
import StreamMonitor from "./pages/StreamMonitor";
import SessionHistory from "./pages/SessionHistory";
import SchedulerPage from "./pages/SchedulerPage";
import SettingsPage from "./pages/SettingsPage";
import AboutPage from "./pages/AboutPage";
import LogViewer from "./pages/LogViewer";

type Page = "dashboard" | "profiles" | "proxies" | "tokens" | "stream" | "history" | "scheduler" | "settings" | "about" | "logs";

interface NavItem {
  id: Page;
  label: string;
  icon: React.ReactNode;
}

const navItems: NavItem[] = [
  { id: "dashboard", label: "Dashboard", icon: <LayoutDashboard size={20} /> },
  { id: "profiles", label: "Profiles", icon: <Users size={20} /> },
  { id: "proxies", label: "Proxies", icon: <Network size={20} /> },
  { id: "tokens", label: "Tokens", icon: <KeyRound size={20} /> },
  { id: "stream", label: "Stream", icon: <MonitorPlay size={20} /> },
  { id: "history", label: "History", icon: <History size={20} /> },
  { id: "scheduler", label: "Scheduler", icon: <CalendarClock size={20} /> },
  { id: "logs", label: "Logs", icon: <ScrollText size={20} /> },
  { id: "settings", label: "Settings", icon: <Settings size={20} /> },
];

export default function App() {
  const [activePage, setActivePage] = useState<Page>("dashboard");
  const [setupComplete, setSetupComplete] = useState(
    () => localStorage.getItem("mongebot-setup-complete") === "true",
  );
  const [showHelp, setShowHelp] = useState(false);

  // ? key opens help overlay
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      const target = e.target as HTMLElement;
      if (target.tagName === "INPUT" || target.tagName === "TEXTAREA") return;
      if (e.key === "?") {
        e.preventDefault();
        setShowHelp((prev) => !prev);
      }
    };
    window.addEventListener("keydown", handler);
    return () => window.removeEventListener("keydown", handler);
  }, []);
  const connected = useConnection();
  const metrics = useMetrics();
  const { stop } = useEngineControl();
  useStreamNotifications();
  useWindowTitle(metrics);
  useKeyboard({
    onToggleEngine: () => {
      if (metrics?.engineState === "running") stop();
    },
    onNavigate: (page) => setActivePage(page as Page),
    metrics,
  });

  // Show onboarding wizard on first run
  if (!setupComplete) {
    return <OnboardingWizard onComplete={() => setSetupComplete(true)} />;
  }

  return (
    <div className="flex h-screen w-screen overflow-hidden">
      {/* Sidebar */}
      <aside className="w-16 bg-gray-900/50 border-r border-gray-800 flex flex-col items-center py-4 gap-1 shrink-0">
        {/* Logo */}
        <div className="mb-4 p-2">
          <Zap size={24} className="text-blue-500" />
        </div>

        {/* Nav Items */}
        {navItems.map((item) => (
          <button
            key={item.id}
            onClick={() => setActivePage(item.id)}
            className={`
              w-11 h-11 rounded-xl flex items-center justify-center
              transition-all duration-150 group relative
              ${
                activePage === item.id
                  ? "bg-blue-600/20 text-blue-400"
                  : "text-gray-500 hover:text-gray-300 hover:bg-gray-800"
              }
            `}
            title={item.label}
          >
            {item.icon}
            {/* Tooltip */}
            <span className="absolute left-full ml-2 px-2 py-1 bg-gray-800 text-gray-200 text-xs rounded-md opacity-0 group-hover:opacity-100 pointer-events-none whitespace-nowrap z-50 transition-opacity">
              {item.label}
            </span>
          </button>
        ))}

        {/* Spacer */}
        <div className="flex-1" />

        {/* Connection status */}
        <div className="mb-2" title={connected ? "Connected" : "Disconnected"}>
          <Radio
            size={16}
            className={connected ? "text-emerald-400 animate-pulse" : "text-red-400"}
          />
        </div>

        {/* About */}
        <button
          onClick={() => setActivePage("about")}
          className={`w-11 h-11 rounded-xl flex items-center justify-center transition-all group relative ${
            activePage === "about" ? "bg-blue-600/20 text-blue-400" : "text-gray-600 hover:text-gray-400"
          }`}
          title="About"
        >
          <Zap size={16} />
        </button>
      </aside>

      {/* Main Content */}
      <main className="flex-1 flex flex-col overflow-hidden">
        <div className="flex-1 overflow-hidden">
          {activePage === "dashboard" && <Dashboard />}
          {activePage === "profiles" && <ProfilesPage />}
          {activePage === "proxies" && <ProxyManager />}
          {activePage === "tokens" && <TokenVault />}
          {activePage === "stream" && <StreamMonitor />}
          {activePage === "history" && <SessionHistory />}
          {activePage === "scheduler" && <SchedulerPage />}
          {activePage === "settings" && <SettingsPage />}
          {activePage === "about" && <AboutPage />}
          {activePage === "logs" && <LogViewer />}
        </div>
        <StatusBar connected={connected} metrics={metrics} />
      </main>

      {/* Connection overlay when disconnected */}
      <ConnectionOverlay connected={connected} />

      {/* Keyboard shortcut help overlay */}
      <KeyboardHelp open={showHelp} onClose={() => setShowHelp(false)} />
    </div>
  );
}
