import { spawnSync } from "node:child_process";
import { mkdtempSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
/* global process -- The CLI test reads the current Node.js process. */
import { describe, expect, it } from "vitest";
import {
  findZeroFunctionCoverage,
  readCoverageFile,
} from "../scripts/check-zero-function-coverage.js";

const webRoot = "/workspace/web";

function functionEntry(name, line) {
  return { name, loc: { start: { line, column: 0 } } };
}

function coverageFile(entries) {
  return {
    path: `${webRoot}/src/example.ts`,
    fnMap: Object.fromEntries(
      entries.map(([id, name, line]) => [id, functionEntry(name, line)]),
    ),
    f: Object.fromEntries(entries.map(([id, , , count = 0]) => [id, count])),
  };
}

describe("findZeroFunctionCoverage", () => {
  it("finds only zero-count functions and sorts normalized locations", () => {
    const coverage = {
      [`${webRoot}/src/zeta.ts`]: coverageFile([
        ["2", "zeta", 30, 0],
        ["1", "alreadyCovered", 10, 2],
      ]),
      [`${webRoot}/src/alpha.ts`]: coverageFile([
        ["2", "anonymous", 8, 0],
        ["1", "named", 8, 0],
      ]),
    };

    expect(findZeroFunctionCoverage(coverage, webRoot)).toEqual([
      { file: "src/alpha.ts", line: 8, name: "anonymous" },
      { file: "src/alpha.ts", line: 8, name: "named" },
      { file: "src/zeta.ts", line: 30, name: "zeta" },
    ]);
  });

  it("returns an empty list when every function ran", () => {
    const coverage = {
      [`${webRoot}/src/example.ts`]: coverageFile([
        ["1", "one", 4, 1],
        ["2", "two", 8, 3],
      ]),
    };

    expect(findZeroFunctionCoverage(coverage, webRoot)).toEqual([]);
  });

  it.each([
    ["empty data", {}],
    ["missing function map", { [`${webRoot}/src/example.ts`]: { f: {} } }],
    [
      "missing count",
      {
        [`${webRoot}/src/example.ts`]: {
          fnMap: { 1: functionEntry("one", 1) },
          f: {},
        },
      },
    ],
    [
      "extra count",
      {
        [`${webRoot}/src/example.ts`]: {
          fnMap: {},
          f: { 1: 0 },
        },
      },
    ],
    [
      "invalid function location",
      {
        [`${webRoot}/src/example.ts`]: {
          fnMap: { 1: { name: "one", loc: {} } },
          f: { 1: 0 },
        },
      },
    ],
  ])("rejects %s", (_, coverage) => {
    expect(() => findZeroFunctionCoverage(coverage, webRoot)).toThrow();
  });
});

describe("readCoverageFile", () => {
  it("rejects a missing coverage file", () => {
    expect(() => readCoverageFile("/definitely/missing/coverage.json")).toThrow(
      /cannot read coverage file/,
    );
  });

  it("rejects malformed JSON", () => {
    const directory = mkdtempSync(join(tmpdir(), "prx-invalid-coverage-"));
    try {
      const coveragePath = join(directory, "coverage-final.json");
      writeFileSync(coveragePath, "not-json");
      expect(() => readCoverageFile(coveragePath)).toThrow(
        /coverage file is not valid JSON/,
      );
    } finally {
      rmSync(directory, { recursive: true, force: true });
    }
  });

  it("fails the CLI for an intentional zero-count fixture", () => {
    const directory = mkdtempSync(join(tmpdir(), "prx-zero-coverage-"));
    try {
      const coveragePath = join(directory, "coverage-final.json");
      const sourceFile = join(process.cwd(), "src/example.ts");
      writeFileSync(
        coveragePath,
        JSON.stringify({
          [sourceFile]: coverageFile([["1", "neverCalled", 12, 0]]),
        }),
      );
      const result = spawnSync(
        process.execPath,
        ["scripts/check-zero-function-coverage.js", coveragePath],
        { cwd: process.cwd(), encoding: "utf8" },
      );

      expect(result.status).toBe(1);
      expect(result.stderr).toContain("src/example.ts:12 neverCalled");
    } finally {
      rmSync(directory, { recursive: true, force: true });
    }
  });
});
