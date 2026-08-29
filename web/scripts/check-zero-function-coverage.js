#!/usr/bin/env node

/* global console, process -- This script runs as a Node.js CLI. */

import { readFileSync } from "node:fs";
import { isAbsolute, relative, resolve, sep } from "node:path";
import { fileURLToPath } from "node:url";

const defaultCoveragePath = "coverage/coverage-final.json";

function isRecord(value) {
  return value !== null && typeof value === "object" && !Array.isArray(value);
}

function relativeWebPath(file, webRoot) {
  const absoluteFile = isAbsolute(file) ? file : resolve(webRoot, file);
  const relativeFile = relative(webRoot, absoluteFile);
  if (
    !relativeFile ||
    relativeFile === ".." ||
    relativeFile.startsWith(`..${sep}`) ||
    isAbsolute(relativeFile)
  ) {
    throw new Error(`coverage file is outside the WebUI root: ${file}`);
  }
  return relativeFile.split(sep).join("/");
}

function compareStrings(left, right) {
  if (left === right) return 0;
  return left < right ? -1 : 1;
}

export function findZeroFunctionCoverage(coverage, webRoot = process.cwd()) {
  if (!isRecord(coverage) || Object.keys(coverage).length === 0)
    throw new Error("coverage data is empty or invalid");

  const zeroFunctions = [];
  let functionCount = 0;
  for (const [file, fileCoverage] of Object.entries(coverage)) {
    if (!isRecord(fileCoverage))
      throw new Error(`coverage entry is invalid: ${file}`);
    const { fnMap, f } = fileCoverage;
    if (!isRecord(fnMap) || !isRecord(f))
      throw new Error(`coverage function data is missing: ${file}`);

    const functionIds = Object.keys(fnMap);
    const countIds = Object.keys(f);
    for (const id of functionIds) {
      if (!Object.hasOwn(f, id))
        throw new Error(`coverage count is missing for ${file} function ${id}`);

      const functionInfo = fnMap[id];
      const count = f[id];
      if (
        !isRecord(functionInfo) ||
        typeof functionInfo.name !== "string" ||
        !isRecord(functionInfo.loc) ||
        !isRecord(functionInfo.loc.start) ||
        typeof functionInfo.loc.start.line !== "number" ||
        !Number.isInteger(functionInfo.loc.start.line) ||
        functionInfo.loc.start.line < 1 ||
        typeof count !== "number" ||
        !Number.isFinite(count) ||
        count < 0
      ) {
        throw new Error(
          `coverage function data is invalid: ${file} function ${id}`,
        );
      }

      functionCount += 1;
      if (count === 0) {
        zeroFunctions.push({
          file: relativeWebPath(file, webRoot),
          line: functionInfo.loc.start.line,
          name: functionInfo.name,
        });
      }
    }
    for (const id of countIds) {
      if (!Object.hasOwn(fnMap, id))
        throw new Error(
          `coverage function is missing for ${file} function ${id}`,
        );
    }
  }

  if (functionCount === 0)
    throw new Error("coverage data contains no functions");

  zeroFunctions.sort(
    (left, right) =>
      compareStrings(left.file, right.file) ||
      left.line - right.line ||
      compareStrings(left.name, right.name),
  );
  return zeroFunctions;
}

export function readCoverageFile(coveragePath) {
  let source;
  try {
    source = readFileSync(coveragePath, "utf8");
  } catch (error) {
    throw new Error(
      `cannot read coverage file ${coveragePath}: ${error instanceof Error ? error.message : String(error)}`,
      { cause: error },
    );
  }
  try {
    return JSON.parse(source);
  } catch (error) {
    throw new Error(
      `coverage file is not valid JSON: ${error instanceof Error ? error.message : String(error)}`,
      { cause: error },
    );
  }
}

export function main(
  coveragePath = defaultCoveragePath,
  webRoot = process.cwd(),
) {
  try {
    const zeroFunctions = findZeroFunctionCoverage(
      readCoverageFile(coveragePath),
      webRoot,
    );
    if (zeroFunctions.length === 0) {
      console.log("Web zero-coverage check passed.");
      return 0;
    }

    console.error("Web functions with zero executions:");
    for (const functionInfo of zeroFunctions)
      console.error(
        `${functionInfo.file}:${functionInfo.line} ${functionInfo.name}`,
      );
    return 1;
  } catch (error) {
    console.error(
      `Web zero-coverage check failed: ${error instanceof Error ? error.message : String(error)}`,
    );
    return 1;
  }
}

if (
  process.argv[1] &&
  fileURLToPath(import.meta.url) === resolve(process.argv[1])
)
  process.exitCode = main(process.argv[2] ?? defaultCoveragePath);
