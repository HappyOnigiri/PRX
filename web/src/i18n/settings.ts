export const supportedLanguages = ["en", "ja"] as const;
export type SupportedLanguage = (typeof supportedLanguages)[number];

export const webUISettingsKey = "prx.webui.settings";
const defaultGraphZoom = 1;
export const minGraphZoom = 0.08;
export const maxGraphZoom = 1.7;

type WebUISettings = {
  language?: SupportedLanguage;
  graphZoom?: number;
};

function isSupportedLanguage(value: unknown): value is SupportedLanguage {
  return supportedLanguages.includes(value as SupportedLanguage);
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
    };
    const settings: WebUISettings = {};
    if (isSupportedLanguage(candidate.language))
      settings.language = candidate.language;
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

export function detectDisplayLanguage(): SupportedLanguage {
  const saved = readWebUISettings().language;
  if (saved) return saved;
  const candidates = navigator.languages?.length
    ? navigator.languages
    : [navigator.language];
  for (const candidate of candidates) {
    const base = candidate.toLowerCase().split("-")[0];
    if (isSupportedLanguage(base)) return base;
  }
  return "en";
}
