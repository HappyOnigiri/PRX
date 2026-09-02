import { featureCategoryIds, type FeatureCategoryId } from "../feature-status";

export const supportedLanguages = ["en", "ja"] as const;
export type SupportedLanguage = (typeof supportedLanguages)[number];

export const themePreferences = ["system", "light", "dark"] as const;
export type ThemePreference = (typeof themePreferences)[number];
export type ResolvedTheme = Exclude<ThemePreference, "system">;

export const webUISettingsKey = "prx.webui.settings";
const defaultGraphZoom = 1;
const defaultFeatureCategory: FeatureCategoryId = "active";
export const minGraphZoom = 0.08;
export const maxGraphZoom = 1.7;

interface WebUISettings {
  language?: SupportedLanguage;
  graphZoom?: number;
  theme?: ThemePreference;
  featureCategory?: FeatureCategoryId;
}

function isSupportedLanguage(value: unknown): value is SupportedLanguage {
  return supportedLanguages.includes(value as SupportedLanguage);
}

function isThemePreference(value: unknown): value is ThemePreference {
  return themePreferences.includes(value as ThemePreference);
}

function isFeatureCategoryId(value: unknown): value is FeatureCategoryId {
  return featureCategoryIds.includes(value as FeatureCategoryId);
}

export function readWebUISettings(): WebUISettings {
  try {
    const value = localStorage.getItem(webUISettingsKey);
    if (!value) return {};
    const parsed: unknown = JSON.parse(value);
    if (!parsed || typeof parsed !== "object") return {};
    const candidate = parsed as {
      language?: unknown;
      graphZoom?: unknown;
      theme?: unknown;
      featureCategory?: unknown;
    };
    const settings: WebUISettings = {};
    if (isSupportedLanguage(candidate.language))
      settings.language = candidate.language;
    if (isThemePreference(candidate.theme)) settings.theme = candidate.theme;
    if (isFeatureCategoryId(candidate.featureCategory))
      settings.featureCategory = candidate.featureCategory;
    if (
      typeof candidate.graphZoom === "number" &&
      Number.isFinite(candidate.graphZoom) &&
      candidate.graphZoom >= minGraphZoom &&
      candidate.graphZoom <= maxGraphZoom
    )
      settings.graphZoom = candidate.graphZoom;
    return settings;
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

// The selected sidebar category is adjusted while working rather than from
// the Settings dialog, so it is stored the way graph zoom is.
export function readFeatureCategory(): FeatureCategoryId {
  return readWebUISettings().featureCategory ?? defaultFeatureCategory;
}

export function writeFeatureCategory(featureCategory: FeatureCategoryId) {
  try {
    const settings = readWebUISettings();
    localStorage.setItem(
      webUISettingsKey,
      JSON.stringify({ ...settings, featureCategory }),
    );
  } catch {
    // The selection still holds for this session when storage is unavailable.
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
