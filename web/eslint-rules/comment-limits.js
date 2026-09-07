// Go 向けの tools/checkcomments と同じく、コメントの塊を短く保つ。
// 上限とこのルールが受け付ける免除マーカーは AGENTS.md に記載している。

// tools/checkcomments が Go のコメントに使うのと同じ East Asian Width W/F の範囲。
const FULL_WIDTH_RANGES = [
  [0x1100, 0x115f],
  [0x231a, 0x231b],
  [0x3008, 0x3009],
  [0x23e9, 0x23ec],
  [0x23f0, 0x23f0],
  [0x23f3, 0x23f3],
  [0x25fd, 0x25fe],
  [0x2614, 0x2615],
  [0x2648, 0x2653],
  [0x267f, 0x267f],
  [0x2693, 0x2693],
  [0x26a1, 0x26a1],
  [0x26aa, 0x26ab],
  [0x26bd, 0x26be],
  [0x26c4, 0x26c5],
  [0x26ce, 0x26ce],
  [0x26d4, 0x26d4],
  [0x26ea, 0x26ea],
  [0x26f2, 0x26f3],
  [0x26f5, 0x26f5],
  [0x26fa, 0x26fa],
  [0x26fd, 0x26fd],
  [0x2705, 0x2705],
  [0x270a, 0x270b],
  [0x2728, 0x2728],
  [0x274c, 0x274c],
  [0x274e, 0x274e],
  [0x2753, 0x2755],
  [0x2757, 0x2757],
  [0x2795, 0x2797],
  [0x27b0, 0x27b0],
  [0x27bf, 0x27bf],
  [0x2b1b, 0x2b1c],
  [0x2b50, 0x2b50],
  [0x2b55, 0x2b55],
  [0x2e80, 0x2e99],
  [0x2e9b, 0x2ef3],
  [0x2f00, 0x2fd5],
  [0x2ff0, 0x2ffb],
  [0x3000, 0x303e],
  [0x3041, 0x3096],
  [0x3099, 0x30ff],
  [0x3105, 0x312f],
  [0x3131, 0x318e],
  [0x3190, 0x31e3],
  [0x31f0, 0x321e],
  [0x3220, 0x3247],
  [0x3250, 0x32ff],
  [0x3300, 0x4dbf],
  [0x4e00, 0xa4c6],
  [0xa960, 0xa97c],
  [0xac00, 0xd7a3],
  [0x8c48, 0xfaff],
  [0xfe10, 0xfe19],
  [0xfe30, 0xfe6b],
  [0xff01, 0xff60],
  [0xffe0, 0xffe6],
  [0x1f004, 0x1f004],
  [0x1f0cf, 0x1f0cf],
  [0x1f18e, 0x1f18e],
  [0x1f191, 0x1f19a],
  [0x1f200, 0x1f202],
  [0x1f210, 0x1f23b],
  [0x1f240, 0x1f248],
  [0x1f250, 0x1f251],
  [0x1f300, 0x1f64f],
  [0x1f680, 0x1f6ff],
  [0x1f900, 0x1f9ff],
  [0x1fa70, 0x1faff],
  [0x20000, 0x3fffd],
];

const MARKER = /^commentlint:allow-long -- (.+)$/u;

// ツール向けディレクティブは文章ではないので、どちらの上限にも数えない。
const DIRECTIVE =
  /^(?:eslint-disable(?:-next-line|-line)?(?:\s|$)|eslint-enable(?:\s|$)|eslint-env(?:\s|$)|@ts-(?:ignore|expect-error|nocheck)(?:\s|$)|prettier-ignore(?:\s|$)|\/\s*<reference\b|[vc]8\s+ignore(?:\s|$)|istanbul\s+ignore(?:\s|$)|@vitest-environment(?:\s|$)|@__PURE__(?:\s|$)|@vite-ignore(?:\s|$)|webpackIgnore(?:\s|:|$))/u;

// これらはブロック形式にしかなく、その中では文章と取り違えようがない。
const BLOCK_DIRECTIVE = /^(?:eslint\s|globals?\s|exported\s)/u;

const LEADING_SPACE = /^[ \t\r]+/u;
const TRAILING_SPACE = /[ \t\r]+$/u;

function trimLeft(text) {
  return text.replace(LEADING_SPACE, "");
}

function isFullWidth(codePoint) {
  return FULL_WIDTH_RANGES.some(
    ([low, high]) => low <= codePoint && codePoint <= high,
  );
}

function displayWidth(text, tabWidth) {
  let width = 0;
  for (const character of text) {
    if (character === "\t") {
      width += tabWidth - (width % tabWidth);
    } else if (isFullWidth(character.codePointAt(0))) {
      width += 2;
    } else {
      width += 1;
    }
  }
  return width;
}

// commentBody はコメント 1 つから区切り記号を取り除き、残った行を返す。
function commentBody(comment) {
  const block = comment.type === "Block";
  const rawLines = comment.value.split("\n");
  const head = trimLeft(comment.value).replace(TRAILING_SPACE, "");
  const directive =
    DIRECTIVE.test(head) || (block && BLOCK_DIRECTIVE.test(head));
  const result = [];
  rawLines.forEach((rawLine, index) => {
    let text = trimLeft(rawLine);
    if (block) {
      text = trimLeft(text.startsWith("*") ? text.slice(1) : text);
      // ブロックコメントの開始・終了だけの行は本文を持たない。
      if (
        text.trim() === "" &&
        (index === 0 || index === rawLines.length - 1)
      ) {
        return;
      }
    }
    result.push({ text, line: comment.loc.start.line + index, directive });
  });
  return result;
}

// commentGroups は、間に他のトークンがない隣接行のコメントを 1 つにまとめる。
function commentGroups(sourceCode) {
  const groups = [];
  let previous = null;
  for (const comment of sourceCode.getAllComments()) {
    if (comment.type !== "Line" && comment.type !== "Block") {
      previous = null;
      continue;
    }
    const adjacent =
      previous !== null &&
      comment.loc.start.line <= previous.loc.end.line + 1 &&
      sourceCode.getTokenBefore(comment, { includeComments: true }) ===
        previous;
    if (adjacent) {
      groups[groups.length - 1].push(comment);
    } else {
      groups.push([comment]);
    }
    previous = comment;
  }
  return groups;
}

function reportAt(context, line, messageId, data) {
  context.report({
    loc: { start: { line, column: 0 }, end: { line, column: 1 } },
    messageId,
    data,
  });
}

// splitBody はマーカー行を本文と分け、見つけたマーカーの違反を報告する。
function splitBody(context, lines, options) {
  const body = [];
  let markers = 0;
  let allowed = false;
  for (const line of lines) {
    if (line.directive) {
      continue;
    }
    if (line.text.includes("commentlint:")) {
      markers += 1;
      const match = MARKER.exec(line.text);
      const valid = match !== null && match[1].trim() !== "";
      if (!valid) {
        reportAt(context, line.line, "markerFormat");
      }
      if (markers > 1) {
        reportAt(context, line.line, "markerDuplicate");
      }
      allowed = allowed || valid;
    } else {
      body.push(line);
    }
    const width = displayWidth(line.text, options.tabWidth);
    if (width > options.maxWidth) {
      reportAt(context, line.line, "width", {
        actual: width,
        max: options.maxWidth,
      });
    }
  }
  while (body.length > 0 && body[0].text.trim() === "") {
    body.shift();
  }
  while (body.length > 0 && body[body.length - 1].text.trim() === "") {
    body.pop();
  }
  return { body, markers, allowed };
}

function checkGroup(context, lines, options) {
  const { body, markers, allowed } = splitBody(context, lines, options);
  if (markers > 0 && body.length === 0) {
    reportAt(context, lines[0].line, "markerEmpty");
  }
  const distinct = new Set(body.map((line) => line.line));
  if (!allowed && distinct.size > options.maxLines) {
    reportAt(context, body[0].line, "lines", {
      actual: distinct.size,
      max: options.maxLines,
    });
  }
}

const rule = {
  meta: {
    type: "problem",
    docs: {
      description:
        "limit the number of lines and the display width of a comment group",
    },
    schema: [
      {
        type: "object",
        properties: {
          maxLines: { type: "integer", minimum: 1 },
          maxWidth: { type: "integer", minimum: 1 },
          tabWidth: { type: "integer", minimum: 1 },
        },
        additionalProperties: false,
      },
    ],
    messages: {
      lines:
        "Comment group spans {{actual}} lines (maximum is {{max}}). Move the rationale into docs/design/ or add `commentlint:allow-long -- <reason>` on its own line.",
      width:
        "Comment line is {{actual}} display columns wide (maximum is {{max}}).",
      markerFormat:
        "Write the waiver as `commentlint:allow-long -- <reason>` on its own comment line.",
      markerDuplicate:
        "A comment group may hold at most one `commentlint:allow-long` marker.",
      markerEmpty:
        "`commentlint:allow-long` needs comment prose to waive; drop the marker.",
    },
  },
  create(context) {
    const options = {
      maxLines: 3,
      maxWidth: 200,
      tabWidth: 8,
      ...context.options[0],
    };
    return {
      Program() {
        for (const group of commentGroups(context.sourceCode)) {
          checkGroup(context, group.flatMap(commentBody), options);
        }
      },
    };
  },
};

export default rule;
