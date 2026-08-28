import { beforeEach, describe, expect, it, vi } from "vitest";
import {
  detectDisplayLanguage,
  readGraphZoom,
  readWebUISettings,
  webUISettingsKey,
  writeDisplayLanguage,
  writeGraphZoom,
} from "../src/i18n/settings";

describe("WebUI settings", () => {
  beforeEach(() => localStorage.clear());

  it("prefers a saved language over the browser language", () => {
    vi.spyOn(window.navigator, "languages", "get").mockReturnValue(["ja-JP"]);
    localStorage.setItem(webUISettingsKey, JSON.stringify({ language: "en" }));
    expect(detectDisplayLanguage()).toBe("en");
  });

  it("uses the browser language and falls back to English", () => {
    vi.spyOn(window.navigator, "languages", "get").mockReturnValue(["ja-JP"]);
    expect(detectDisplayLanguage()).toBe("ja");
    vi.spyOn(window.navigator, "languages", "get").mockReturnValue(["fr-FR"]);
    expect(detectDisplayLanguage()).toBe("en");
  });

  it("ignores malformed Local Storage data", () => {
    localStorage.setItem(webUISettingsKey, "not-json");
    expect(readWebUISettings()).toEqual({});
  });

  it("restores a saved graph zoom", () => {
    localStorage.setItem(
      webUISettingsKey,
      JSON.stringify({ language: "ja", graphZoom: 0.72 }),
    );
    expect(readGraphZoom()).toBe(0.72);
  });

  it("preserves other WebUI settings when updating language or zoom", () => {
    writeGraphZoom(0.64);
    writeDisplayLanguage("ja");
    expect(readWebUISettings()).toEqual({ language: "ja", graphZoom: 0.64 });
  });

  it("uses the default graph zoom when the saved value is invalid", () => {
    localStorage.setItem(webUISettingsKey, JSON.stringify({ graphZoom: 20 }));
    expect(readGraphZoom()).toBe(1);
  });
});
