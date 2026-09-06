import { beforeEach, describe, expect, it, vi } from "vitest";
import {
  detectDisplayLanguage,
  readCollapsedProjects,
  readGraphZoom,
  readThemePreference,
  readWebUISettings,
  resolveThemePreference,
  webUISettingsKey,
  writeCollapsedProjects,
  writeDisplayLanguage,
  writeGraphZoom,
  writeThemePreference,
} from "../src/i18n/settings";

describe("WebUI settings", () => {
  beforeEach(() => {
    localStorage.clear();
  });

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
    writeThemePreference("dark");
    writeGraphZoom(0.64);
    writeDisplayLanguage("ja");
    writeCollapsedProjects(["P-1"]);
    expect(readWebUISettings()).toEqual({
      language: "ja",
      graphZoom: 0.64,
      theme: "dark",
      collapsedProjects: ["P-1"],
    });
  });

  // Only the collapsed rows are stored, so anything the file cannot vouch for
  // has to read back as "nothing is collapsed" and leave the tree expanded.
  it("restores the collapsed sidebar rows and rejects a malformed list", () => {
    expect(readCollapsedProjects()).toEqual([]);
    writeCollapsedProjects(["P-2", "unassigned"]);
    expect(readCollapsedProjects()).toEqual(["P-2", "unassigned"]);
    for (const collapsedProjects of ["P-1", [1], [""], { id: "P-1" }]) {
      localStorage.setItem(
        webUISettingsKey,
        JSON.stringify({ collapsedProjects }),
      );
      expect(readCollapsedProjects()).toEqual([]);
    }
  });

  it("keeps the session usable when Local Storage refuses a write", () => {
    const setItem = vi
      .spyOn(Storage.prototype, "setItem")
      .mockImplementation(() => {
        throw new Error("quota exceeded");
      });
    expect(() => {
      writeCollapsedProjects(["P-1"]);
    }).not.toThrow();
    setItem.mockRestore();
  });

  it("uses the default graph zoom when the saved value is invalid", () => {
    localStorage.setItem(webUISettingsKey, JSON.stringify({ graphZoom: 20 }));
    expect(readGraphZoom()).toBe(1);
  });

  it("defaults to system theme and ignores an invalid saved theme", () => {
    expect(readThemePreference()).toBe("system");
    localStorage.setItem(webUISettingsKey, JSON.stringify({ theme: "sepia" }));
    expect(readThemePreference()).toBe("system");
  });

  it("resolves system theme to light unless dark is explicitly preferred", () => {
    expect(resolveThemePreference("system", false)).toBe("light");
    expect(resolveThemePreference("system", true)).toBe("dark");
    expect(resolveThemePreference("light", true)).toBe("light");
    expect(resolveThemePreference("dark", false)).toBe("dark");
  });
});
