// Theme management with localStorage persistence and accent color support.

export type Theme = "dark" | "light";
export type AccentColor = "blue" | "emerald" | "violet" | "rose" | "amber" | "cyan";

interface UIPreferences {
  theme: Theme;
  accentColor: AccentColor;
  compactMode: boolean;
  showCharts: boolean;
}

const STORAGE_KEY = "mongebot-ui-prefs";

const defaults: UIPreferences = {
  theme: "dark",
  accentColor: "blue",
  compactMode: false,
  showCharts: true,
};

// Accent color CSS variable mappings
const accentMap: Record<AccentColor, { primary: string; hover: string; ring: string; bg: string }> = {
  blue:    { primary: "59 130 246",  hover: "96 165 250",  ring: "59 130 246",  bg: "59 130 246" },
  emerald: { primary: "16 185 129",  hover: "52 211 153",  ring: "16 185 129",  bg: "16 185 129" },
  violet:  { primary: "139 92 246",  hover: "167 139 250", ring: "139 92 246",  bg: "139 92 246" },
  rose:    { primary: "244 63 94",   hover: "251 113 133", ring: "244 63 94",   bg: "244 63 94" },
  amber:   { primary: "245 158 11",  hover: "252 191 73",  ring: "245 158 11",  bg: "245 158 11" },
  cyan:    { primary: "6 182 212",   hover: "34 211 238",  ring: "6 182 212",   bg: "6 182 212" },
};

// loadPrefs reads saved preferences from localStorage.
export function loadPrefs(): UIPreferences {
  try {
    const saved = localStorage.getItem(STORAGE_KEY);
    if (saved) {
      return { ...defaults, ...JSON.parse(saved) };
    }
  } catch {
    // Ignore parse errors
  }
  return { ...defaults };
}

// savePrefs persists preferences to localStorage.
export function savePrefs(prefs: UIPreferences): void {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(prefs));
  } catch {
    // Storage might be unavailable in Tauri context
  }
}

// applyTheme sets CSS variables and body class for the active theme.
export function applyTheme(prefs: UIPreferences): void {
  const root = document.documentElement;
  const body = document.body;

  // Theme class on both root and body
  root.classList.remove("light", "dark");
  body.classList.remove("light", "dark");
  root.classList.add(prefs.theme);
  body.classList.add(prefs.theme);

  // Light theme overrides
  if (prefs.theme === "light") {
    root.style.setProperty("--bg-primary", "255 255 255");
    root.style.setProperty("--bg-secondary", "249 250 251");
    root.style.setProperty("--text-primary", "17 24 39");
    root.style.setProperty("--text-secondary", "107 114 128");
    root.style.setProperty("--border-color", "229 231 235");
    body.style.backgroundColor = "rgb(249 250 251)";
    body.style.color = "rgb(17 24 39)";
  } else {
    root.style.removeProperty("--bg-primary");
    root.style.removeProperty("--bg-secondary");
    root.style.removeProperty("--text-primary");
    root.style.removeProperty("--text-secondary");
    root.style.removeProperty("--border-color");
    body.style.backgroundColor = "rgb(3 7 18)";
    body.style.color = "rgb(243 244 246)";
  }

  // Accent color
  const accent = accentMap[prefs.accentColor] || accentMap.blue;
  root.style.setProperty("--accent-primary", accent.primary);
  root.style.setProperty("--accent-hover", accent.hover);
  root.style.setProperty("--accent-ring", accent.ring);

  // Compact mode
  if (prefs.compactMode) {
    root.style.setProperty("--spacing-scale", "0.85");
  } else {
    root.style.removeProperty("--spacing-scale");
  }
}

// Available accent colors for the picker UI.
export const availableAccents: { key: AccentColor; label: string; swatch: string }[] = [
  { key: "blue", label: "Blue", swatch: "bg-blue-500" },
  { key: "emerald", label: "Green", swatch: "bg-emerald-500" },
  { key: "violet", label: "Purple", swatch: "bg-violet-500" },
  { key: "rose", label: "Rose", swatch: "bg-rose-500" },
  { key: "amber", label: "Amber", swatch: "bg-amber-500" },
  { key: "cyan", label: "Cyan", swatch: "bg-cyan-500" },
];
