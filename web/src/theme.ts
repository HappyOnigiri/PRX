import {
  readThemePreference,
  resolveThemePreference,
  type ThemePreference,
  writeThemePreference,
} from "./i18n/settings";

const darkSchemeQuery = "(prefers-color-scheme: dark)";
const themeColors = { light: "#f5f6f8", dark: "#191b1f" } as const;
interface ThemeWindow {
  matchMedia?: Window["matchMedia"];
}

const themeWindow = window as ThemeWindow;

function prefersDark() {
  const mediaQuery = themeWindow.matchMedia;
  return mediaQuery ? mediaQuery(darkSchemeQuery).matches : false;
}

function applyThemePreference(theme: ThemePreference) {
  if (theme === "system") delete document.documentElement.dataset["theme"];
  else document.documentElement.dataset["theme"] = theme;

  const resolved = resolveThemePreference(theme, prefersDark());
  document
    .querySelector('meta[name="theme-color"]')
    ?.setAttribute("content", themeColors[resolved]);
}

export function setDisplayTheme(theme: ThemePreference) {
  writeThemePreference(theme);
  applyThemePreference(theme);
}

applyThemePreference(readThemePreference());

themeWindow.matchMedia?.(darkSchemeQuery).addEventListener("change", () => {
  if (readThemePreference() === "system") applyThemePreference("system");
});
