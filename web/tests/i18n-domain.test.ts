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

  it("keeps an unexpected error message in its original form", () => {
    expect(formatError(new Error("unexpected backend failure"), t)).toBe(
      "unexpected backend failure",
    );
  });
});
