import { ConnectError } from "@connectrpc/connect";
import { Code } from "@connectrpc/connect";
import { describe, expect, it } from "vitest";
import { DomainErrorCode, ErrorDetailSchema } from "../src/gen/prx/v1/prx_pb";
import i18n from "../src/i18n";
import { formatError } from "../src/i18n/domain";

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
