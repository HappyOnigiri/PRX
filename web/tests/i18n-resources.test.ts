import { describe, expect, it } from "vitest";
import { resources } from "../src/i18n/resources";
import { supportedLanguages } from "../src/i18n/settings";

function flattenKeys(value: object, prefix = ""): string[] {
  return Object.entries(value).flatMap(([key, child]) => {
    const nested: unknown = child;
    return nested && typeof nested === "object"
      ? flattenKeys(nested, `${prefix}${key}.`)
      : [`${prefix}${key}`];
  });
}

describe("translation resources", () => {
  // Only the English resource types the t() calls, so a key missing from another
  // language passes type checking and silently falls back at runtime.
  it("defines the same keys in every supported language", () => {
    const english = flattenKeys(resources.en.translation).sort();
    for (const language of supportedLanguages) {
      expect({
        language,
        keys: flattenKeys(resources[language].translation).sort(),
      }).toEqual({ language, keys: english });
    }
  });
});
