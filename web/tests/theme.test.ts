import { beforeEach, describe, expect, it, vi } from "vitest";
import { webUISettingsKey } from "../src/i18n/settings";
import { setDisplayTheme } from "../src/theme";

describe("display theme", () => {
  beforeEach(() => {
    localStorage.clear();
    delete document.documentElement.dataset["theme"];
    document.head.innerHTML = '<meta name="theme-color" content="#f5f6f8">';
    vi.unstubAllGlobals();
  });

  it("applies and stores an explicit theme", () => {
    setDisplayTheme("dark");

    expect(document.documentElement.dataset["theme"]).toBe("dark");
    expect(document.querySelector('meta[name="theme-color"]')).toHaveAttribute(
      "content",
      "#191b1f",
    );
    const storedSettings = localStorage.getItem(webUISettingsKey);
    if (storedSettings === null)
      throw new Error("The theme preference was not stored.");
    expect(JSON.parse(storedSettings)).toEqual({
      theme: "dark",
    });
  });

  it("removes the override and resolves system dark preference", () => {
    vi.stubGlobal("matchMedia", vi.fn().mockReturnValue({ matches: true }));
    setDisplayTheme("light");
    setDisplayTheme("system");

    expect(document.documentElement).not.toHaveAttribute("data-theme");
    expect(document.querySelector('meta[name="theme-color"]')).toHaveAttribute(
      "content",
      "#191b1f",
    );
  });
});
