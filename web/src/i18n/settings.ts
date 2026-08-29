export const supportedLanguages = ["en", "ja"] as const;
export type SupportedLanguage = (typeof supportedLanguages)[number];

export const themePreferences = ["system", "light", "dark"] as const;
export type ThemePreference = (typeof themePreferences)[number];
export type ResolvedTheme = Exclude<ThemePreference, "system">;

export const webUISettingsKey = "prx.webui.settings";
const defaultGraphZoom = 1;
export const minGraphZoom = 0.08;
export const maxGraphZoom = 1.7;

interface WebUISettings {
  language?: SupportedLanguage;
  graphZoom?: number;
  theme?: ThemePreference;
}

function isSupportedLanguage(value: unknown): value is SupportedLanguage {
  return supportedLanguages.includes(value as SupportedLanguage);
}

function isThemePreference(value: unknown): value is ThemePreference {
  return themePreferences.includes(value as ThemePreference);
}

function isSettingsRecord(value: unknown): value is Record<string, unknown> {
  return value !== null && typeof value === "object";
}

function readLanguage(value: unknown): Pick<WebUISettings, "language"> {
  return isSupportedLanguage(value) ? { language: value } : {};
}

function readTheme(value: unknown): Pick<WebUISettings, "theme"> {
  return isThemePreference(value) ? { theme: value } : {};
}

function readGraphZoomValue(value: unknown): Pick<WebUISettings, "graphZoom"> {
  if (
    typeof value === "number" &&
    Number.isFinite(value) &&
    value >= minGraphZoom &&
    value <= maxGraphZoom
  )
    return { graphZoom: value };
  return {};
}

function parseWebUISettings(value: string): WebUISettings {
  const parsed: unknown = JSON.parse(value);
  if (!isSettingsRecord(parsed)) return {};
  return {
    ...readLanguage(parsed["language"]),
    ...readTheme(parsed["theme"]),
    ...readGraphZoomValue(parsed["graphZoom"]),
  };
}

export function readWebUISettings(): WebUISettings {
  try {
    const value = localStorage.getItem(webUISettingsKey);
    if (!value) return {};
    return parseWebUISettings(value);
  } catch {
    return {};
  }
}

export function readGraphZoom() {
  return readWebUISettings().graphZoom ?? defaultGraphZoom;
}

export function writeGraphZoom(graphZoom: number) {
  if (
    !Number.isFinite(graphZoom) ||
    graphZoom < minGraphZoom ||
    graphZoom > maxGraphZoom
  )
    return;
  try {
    const settings = readWebUISettings();
    localStorage.setItem(
      webUISettingsKey,
      JSON.stringify({ ...settings, graphZoom }),
    );
  } catch {
    // The zoom still changes for this session when storage is unavailable.
  }
}

export function writeDisplayLanguage(language: SupportedLanguage) {
  try {
    const settings = readWebUISettings();
    localStorage.setItem(
      webUISettingsKey,
      JSON.stringify({ ...settings, language }),
    );
  } catch {
    // The language still changes for this session when storage is unavailable.
  }
}

export function readThemePreference(): ThemePreference {
  return readWebUISettings().theme ?? "system";
}

export function resolveThemePreference(
  theme: ThemePreference,
  prefersDark: boolean,
): ResolvedTheme {
  return theme === "system" ? (prefersDark ? "dark" : "light") : theme;
}

export function writeThemePreference(theme: ThemePreference) {
  try {
    const settings = readWebUISettings();
    localStorage.setItem(
      webUISettingsKey,
      JSON.stringify({ ...settings, theme }),
    );
  } catch {
    // The theme still changes for this session when storage is unavailable.
  }
}

export function detectDisplayLanguage(): SupportedLanguage {
  const saved = readWebUISettings().language;
  if (saved) return saved;
  const candidates = navigator.languages.length
    ? navigator.languages
    : [navigator.language];
  for (const candidate of candidates) {
    const base = candidate.toLowerCase().split("-")[0];
    if (isSupportedLanguage(base)) return base;
  }
  return "en";
}
