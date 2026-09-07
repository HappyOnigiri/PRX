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
  // t() の型付けは英語リソースだけを見るので、他言語で欠けたキーは型検査を
  // 通り、実行時に黙ってフォールバックする。
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
