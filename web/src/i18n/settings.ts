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
  collapsedProjects?: string[];
  hideCompletedTasks?: boolean;
}

function isSupportedLanguage(value: unknown): value is SupportedLanguage {
  return supportedLanguages.includes(value as SupportedLanguage);
}

function isThemePreference(value: unknown): value is ThemePreference {
  return themePreferences.includes(value as ThemePreference);
}

function isCollapsedProjects(value: unknown): value is string[] {
  return (
    Array.isArray(value) &&
    value.every((entry) => typeof entry === "string" && entry !== "")
  );
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
      collapsedProjects?: unknown;
      hideCompletedTasks?: unknown;
    };
    const settings: WebUISettings = {};
    if (typeof candidate.hideCompletedTasks === "boolean")
      settings.hideCompletedTasks = candidate.hideCompletedTasks;
    if (isSupportedLanguage(candidate.language))
      settings.language = candidate.language;
    if (isThemePreference(candidate.theme)) settings.theme = candidate.theme;
    if (isCollapsedProjects(candidate.collapsedProjects))
      settings.collapsedProjects = candidate.collapsedProjects;
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
    // ストレージが使えなくても、このセッション中はズームが変わる。
  }
}

// サイドバーの project の折りたたみは設定ダイアログではなく作業中に切り替えるの
// で、グラフのズームと同じ方法で保存する。折りたたみ中の ID だけを持つため、未知
// の project は既定で展開される。
export function readCollapsedProjects(): string[] {
  return readWebUISettings().collapsedProjects ?? [];
}

export function writeCollapsedProjects(collapsedProjects: string[]) {
  try {
    const settings = readWebUISettings();
    localStorage.setItem(
      webUISettingsKey,
      JSON.stringify({ ...settings, collapsedProjects }),
    );
  } catch {
    // ストレージが使えなくても、このセッション中は行が折りたたまれる。
  }
}

// 完了 task の非表示は設定ダイアログではなくグラフを見ながら切り替えるので、
// グラフのズームと同じ方法で保存する。未保存ならグラフは全 task を表示する。
export function readHideCompletedTasks(): boolean {
  return readWebUISettings().hideCompletedTasks ?? false;
}

export function writeHideCompletedTasks(hideCompletedTasks: boolean) {
  try {
    const settings = readWebUISettings();
    localStorage.setItem(
      webUISettingsKey,
      JSON.stringify({ ...settings, hideCompletedTasks }),
    );
  } catch {
    // ストレージが使えなくても、このセッション中は絞り込みが効く。
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
    // ストレージが使えなくても、このセッション中は言語が変わる。
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
    // ストレージが使えなくても、このセッション中はテーマが変わる。
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
