import { RuleTester } from "eslint";
import { afterAll, describe, it } from "vitest";
import rule from "../eslint-rules/comment-limits.js";

// vitest does not inject globals here, so RuleTester needs the hooks handed to it.
RuleTester.describe = describe;
RuleTester.it = it;
RuleTester.itOnly = it.only;
RuleTester.afterAll = afterAll;

const ruleTester = new RuleTester({
  languageOptions: {
    ecmaVersion: "latest",
    sourceType: "module",
    parserOptions: { ecmaFeatures: { jsx: true } },
  },
});

const repeat = (text, count) => text.repeat(count);

const valid = [
  { name: "three", code: "// a\n// b\n// c\nconst x = 1;" },
  { name: "block", code: "/*\n * a\n * b\n * c\n */\nconst x = 1;" },
  { name: "same physical line", code: "/* a */ /* b */ /* c */ /* d */" },
  {
    name: "separated by a blank line",
    code: "// a\n// b\n\n// c\n// d\nconst x = 1;",
  },
  { name: "separated by code", code: "const x = 1; // a\nconst y = 2; // b" },
  { name: "ascii boundary", code: `// ${repeat("a", 200)}` },
  { name: "japanese boundary", code: `// ${repeat("あ", 100)}` },
  { name: "tab boundary", code: `// ${repeat("a", 192)}\t` },
  { name: "indent", code: `//\t\t${repeat("a", 200)}` },
  {
    name: "waived",
    code: "// a\n// b\n// c\n// d\n// commentlint:allow-long -- states the contract",
  },
  {
    name: "line directives",
    code: "// eslint-disable-next-line no-eval -- reason\n// @ts-expect-error -- reason\n// v8 ignore next\n// a\n// b\n// c\nconst x = 1;",
  },
  {
    name: "reference directive",
    code: '/// <reference types="vite/client" />',
  },
  {
    name: "block directives",
    code: '/* eslint no-eval: "error" -- reason */\n/* global window */\n// a\n// b\n// c\nconst x = 1;',
  },
  {
    name: "jsx comment",
    code: "const node = <div>{/* a */}</div>;",
  },
  {
    name: "hashbang",
    code: "#!/usr/bin/env node\n// a\n// b\n// c\nconst x = 1;",
  },
  { name: "custom threshold", code: "// a\n// b", options: [{ maxLines: 2 }] },
  {
    name: "custom tab width",
    code: "// a\t",
    options: [{ maxWidth: 4, tabWidth: 4 }],
  },
];

const invalid = [
  {
    name: "four",
    code: "// a\n// b\n// c\n// d",
    errors: [{ messageId: "lines" }],
  },
  {
    name: "bare blank line does not split a group",
    code: "// a\n//\n// b\n// c",
    errors: [{ messageId: "lines" }],
  },
  {
    name: "block four",
    code: "/* a\n b\n c\n d */",
    errors: [{ messageId: "lines" }],
  },
  {
    name: "directive does not raise the limit",
    code: "// eslint-disable-next-line no-eval -- reason\n// a\n// b\n// c\n// d\nconst x = 1;",
    errors: [{ messageId: "lines" }],
  },
  {
    name: "ascii wide",
    code: `// ${repeat("a", 201)}`,
    errors: [{ messageId: "width", data: { actual: 201, max: 200 } }],
  },
  {
    name: "japanese wide",
    code: `// ${repeat("あ", 100)}a`,
    errors: [{ messageId: "width" }],
  },
  {
    name: "tab wide",
    code: `// ${repeat("a", 200)}\t`,
    errors: [{ messageId: "width" }],
  },
  {
    name: "waiver keeps the width limit",
    code: `// ${repeat("a", 201)}\n// commentlint:allow-long -- reason`,
    errors: [{ messageId: "width" }],
  },
  {
    name: "marker is measured too",
    code: `// a\n// commentlint:allow-long -- ${repeat("a", 200)}`,
    errors: [{ messageId: "width" }],
  },
  {
    name: "reason missing",
    code: "// a\n// commentlint:allow-long -- ",
    errors: [{ messageId: "markerFormat" }],
  },
  {
    name: "blank unicode reason",
    code: "// a\n// commentlint:allow-long -- 　",
    errors: [{ messageId: "markerFormat" }],
  },
  {
    name: "unknown marker",
    code: "// a\n// commentlint:ignore -- reason",
    errors: [{ messageId: "markerFormat" }],
  },
  {
    name: "inline marker",
    code: "// a commentlint:allow-long -- reason",
    errors: [{ messageId: "markerFormat" }, { messageId: "markerEmpty" }],
  },
  {
    name: "duplicate marker",
    code: "// a\n// commentlint:allow-long -- reason\n// commentlint:allow-long -- reason",
    errors: [{ messageId: "markerDuplicate" }],
  },
  {
    name: "marker only",
    code: "// commentlint:allow-long -- reason",
    errors: [{ messageId: "markerEmpty" }],
  },
  {
    name: "custom threshold",
    code: "// a\n// b\n// c",
    options: [{ maxLines: 2 }],
    errors: [{ messageId: "lines" }],
  },
  {
    name: "custom width",
    code: "// aaaa",
    options: [{ maxWidth: 3 }],
    errors: [{ messageId: "width" }],
  },
  {
    name: "default tab width",
    code: "// a\t",
    options: [{ maxWidth: 4 }],
    errors: [{ messageId: "width", data: { actual: 8, max: 4 } }],
  },
];

ruleTester.run("comment-limits", rule, { valid, invalid });
