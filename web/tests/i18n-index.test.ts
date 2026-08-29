import { beforeEach, describe, expect, it } from "vitest";
import i18n, { setDisplayLanguage } from "../src/i18n";
import { readWebUISettings } from "../src/i18n/settings";

describe("display language", () => {
  beforeEach(async () => {
    localStorage.clear();
    await setDisplayLanguage("en");
  });

  it("stores the selected language and updates the document", async () => {
    await setDisplayLanguage("ja");

    expect(i18n.resolvedLanguage).toBe("ja");
    expect(document.documentElement.lang).toBe("ja");
    expect(document.title).toBe("PRX — 依存関係コントロール");
    expect(readWebUISettings()).toEqual({ language: "ja" });
  });
});
