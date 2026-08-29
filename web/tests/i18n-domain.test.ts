import { Code, ConnectError } from "@connectrpc/connect";
import { describe, expect, it } from "vitest";
import {
  BlockedReasonCode,
  DocumentKind,
  DomainErrorCode,
  ErrorDetailSchema,
  FeatureStatus,
  PullRequestDisplayState,
  TaskDisplayState,
  TaskKind,
  TaskStatus,
} from "../src/gen/prx/v1/prx_pb";
import i18n from "../src/i18n";
import {
  blockedReasonKeys,
  displayStateKeys,
  documentKindKeys,
  errorKeys,
  featureStatusKeys,
  formatError,
  pullRequestDisplayStateKeys,
  pullRequestDisplayStateToken,
  taskKindKeys,
  taskStatusKeys,
} from "../src/i18n/domain";
import { resources } from "../src/i18n/resources";
import {
  supportedLanguages,
  type SupportedLanguage,
} from "../src/i18n/settings";

function enumValues(value: object): number[] {
  return Object.values(value).filter(
    (entry): entry is number => typeof entry === "number",
  );
}

function translationValue(language: SupportedLanguage, key: string): unknown {
  let value: unknown = resources[language].translation;
  for (const part of key.split(".")) {
    if (!value || typeof value !== "object") return undefined;
    value = (value as Record<string, unknown>)[part];
  }
  return value;
}

const translationTables = [
  {
    name: "feature statuses",
    values: enumValues(FeatureStatus),
    keys: featureStatusKeys as unknown as Record<number, string>,
  },
  {
    name: "task statuses",
    values: enumValues(TaskStatus),
    keys: taskStatusKeys as unknown as Record<number, string>,
  },
  {
    name: "task kinds",
    values: enumValues(TaskKind),
    keys: taskKindKeys as unknown as Record<number, string>,
  },
  {
    name: "document kinds",
    values: enumValues(DocumentKind),
    keys: documentKindKeys as unknown as Record<number, string>,
  },
  {
    name: "task display states",
    values: enumValues(TaskDisplayState),
    keys: displayStateKeys as unknown as Record<number, string>,
  },
  {
    name: "pull request display states",
    values: enumValues(PullRequestDisplayState),
    keys: pullRequestDisplayStateKeys as unknown as Record<number, string>,
  },
  {
    name: "blocked reason codes",
    values: enumValues(BlockedReasonCode),
    keys: blockedReasonKeys as unknown as Record<number, string>,
  },
  {
    name: "domain error codes",
    values: enumValues(DomainErrorCode),
    keys: errorKeys as unknown as Record<number, string>,
  },
];

describe("domain translation mappings", () => {
  it.each(translationTables)(
    "$name has a label in every supported language for every enum value",
    ({ values, keys }) => {
      for (const value of values) {
        const key = keys[value];
        expect(key, `missing mapping for enum value ${value}`).toBeDefined();
        if (key === undefined) continue;
        for (const language of supportedLanguages) {
          const label = translationValue(language, key);
          expect(label, `${language} ${key}`).toEqual(expect.any(String));
          expect(label, `${language} ${key}`).not.toBe(key);
        }
      }
    },
  );
});

describe("localized RPC errors", () => {
  const t = i18n.getFixedT("ja");

  it("formats a known domain error from structured details", () => {
    const error = new ConnectError(
      "cycle would be introduced",
      Code.FailedPrecondition,
      undefined,
      [
        {
          desc: ErrorDetailSchema,
          value: { code: DomainErrorCode.CYCLE, path: ["API", "UI", "API"] },
        },
      ],
    );
    expect(formatError(error, t)).toBe(
      "この依存関係を追加すると循環します: API → UI → API",
    );
  });

  it("resolves cycle path identifiers to task titles", () => {
    const titles = new Map([
      ["task-a", "API"],
      ["task-b", "UI"],
    ]);
    const error = new ConnectError(
      "cycle would be introduced",
      Code.FailedPrecondition,
      undefined,
      [
        {
          desc: ErrorDetailSchema,
          value: {
            code: DomainErrorCode.CYCLE,
            path: ["task-a", "task-b", "task-a"],
          },
        },
      ],
    );
    expect(formatError(error, t, (id) => titles.get(id))).toBe(
      "この依存関係を追加すると循環します: API → UI → API",
    );
  });

  it("keeps unresolvable cycle path identifiers as they are", () => {
    const error = new ConnectError(
      "cycle would be introduced",
      Code.FailedPrecondition,
      undefined,
      [
        {
          desc: ErrorDetailSchema,
          value: { code: DomainErrorCode.CYCLE, path: ["task-a", "task-a"] },
        },
      ],
    );
    expect(formatError(error, t, () => undefined)).toBe(
      "この依存関係を追加すると循環します: task-a → task-a",
    );
  });

  it("keeps an unexpected error message in its original form", () => {
    expect(formatError(new Error("unexpected backend failure"), t)).toBe(
      "unexpected backend failure",
    );
  });
});

describe("display state style tokens", () => {
  it("maps pull request enum names to CSS state tokens", () => {
    expect(
      pullRequestDisplayStateToken(PullRequestDisplayState.REVIEW_WAITING),
    ).toBe("review_waiting");
    expect(
      pullRequestDisplayStateToken(PullRequestDisplayState.UNSPECIFIED),
    ).toBe("unspecified");
  });
});
