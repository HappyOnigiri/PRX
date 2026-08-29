import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { webUISettingsKey } from "../src/i18n/settings";

describe("system theme changes", () => {
  const mediaQuery = {
    matches: false,
    addEventListener: vi.fn<(type: string, listener: () => void) => void>(),
    removeEventListener: vi.fn<(type: string, listener: () => void) => void>(),
  };

  beforeEach(() => {
    vi.resetModules();
    vi.clearAllMocks();
    localStorage.clear();
    document.head.innerHTML = '<meta name="theme-color" content="#f5f6f8">';
    delete document.documentElement.dataset["theme"];
    mediaQuery.matches = false;
    vi.stubGlobal(
      "matchMedia",
      vi.fn(() => mediaQuery),
    );
    localStorage.setItem(webUISettingsKey, JSON.stringify({ theme: "system" }));
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("reapplies system theme colors when the media query changes", async () => {
    await import("../src/theme");
    expect(mediaQuery.addEventListener).toHaveBeenCalledWith(
      "change",
      expect.any(Function),
    );
    const listener = mediaQuery.addEventListener.mock.calls[0]?.[1];
    if (typeof listener !== "function")
      throw new Error("theme listener missing");

    mediaQuery.matches = true;
    listener();
    expect(document.querySelector('meta[name="theme-color"]')).toHaveAttribute(
      "content",
      "#191b1f",
    );
  });
});
