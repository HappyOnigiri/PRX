import {
  TaskBlockLabel,
  type Feature,
  type PullRequest,
  type Snapshot,
  type Task,
} from "./gen/prx/v1/prx_pb";
import {
  pullRequestDisplayStateToken,
  taskDisplayStateToken,
} from "./i18n/domain";

// task-status はステータスだけを受け付ける。進行を妨げている事情は別の軸なので
// block: で絞る。docs/design/webui.md を参照。
const taskStatusValues = [
  "ready",
  "not-started",
  "designing",
  "designed",
  "in-progress",
  "implemented",
  "in-review",
  "approved",
  "merged",
  "completed",
  "closed",
  "unknown",
] as const;

const blockValues = [
  "dependency",
  "conflict",
  "changes-requested",
  "ci-failed",
] as const;

const githubStatusValues = [
  "merged",
  "closed",
  "draft",
  "conflict",
  "changes-requested",
  "approved",
  "review-waiting",
  "open",
  "unknown",
  "error",
] as const;

type TaskStatusValue = (typeof taskStatusValues)[number];
type GithubStatusValue = (typeof githubStatusValues)[number];
type BlockValue = (typeof blockValues)[number];

type TaskSearchQualifier =
  | { type: "task-status"; value: TaskStatusValue }
  | { type: "github-status"; value: GithubStatusValue }
  | { type: "block"; value: BlockValue };

export interface TaskSearchQuery {
  qualifiers: TaskSearchQualifier[];
  terms: string[];
}

export type TaskSearchParseError =
  | {
      type: "invalid-qualifier";
      key: "task-status" | "github-status" | "block";
      value: string;
    }
  | { type: "unterminated-quote" };

export interface TaskSearchParseResult extends TaskSearchQuery {
  error?: TaskSearchParseError;
}

export interface TaskSearchResult {
  task: Task;
  feature: Feature;
  pullRequest?: PullRequest;
}

interface SearchToken {
  value: string;
  quoted: boolean;
}

export function parseTaskSearch(input: string): TaskSearchParseResult {
  const tokenized = tokenize(input);
  if (tokenized.error)
    return { qualifiers: [], terms: [], error: tokenized.error };

  const qualifiers: TaskSearchQualifier[] = [];
  const terms: string[] = [];
  for (const token of tokenized.tokens) {
    const separator = token.value.indexOf(":");
    const key =
      separator < 0 ? "" : token.value.slice(0, separator).toLowerCase();
    const value =
      separator < 0 ? "" : token.value.slice(separator + 1).toLowerCase();
    if (
      !token.quoted &&
      (key === "task-status" || key === "github-status" || key === "block")
    ) {
      if (key === "task-status" && isTaskStatusValue(value))
        qualifiers.push({ type: key, value });
      else if (key === "github-status" && isGithubStatusValue(value))
        qualifiers.push({ type: key, value });
      else if (key === "block" && isBlockValue(value))
        qualifiers.push({ type: key, value });
      else
        return {
          qualifiers,
          terms,
          error: { type: "invalid-qualifier", key, value },
        };
      continue;
    }
    if (token.value) terms.push(token.value);
  }
  return { qualifiers, terms };
}

export function filterTaskSearchResults(
  snapshot: Snapshot,
  query: TaskSearchQuery,
): TaskSearchResult[] {
  const featuresByID = new Map(
    snapshot.features.map((feature) => [feature.id, feature]),
  );
  const pullRequestsByTaskID = new Map(
    snapshot.pullRequests.map((pullRequest) => [
      pullRequest.taskId,
      pullRequest,
    ]),
  );
  return snapshot.tasks.flatMap((task) => {
    const feature = featuresByID.get(task.featureId);
    if (
      // read-only の feature は、自身がアーカイブ済みでも、アーカイブ済み
      // project の配下にあっても、作業対象外。
      !feature ||
      feature.readOnly ||
      !matchesQualifiers(
        task,
        pullRequestsByTaskID.get(task.id),
        query.qualifiers,
      )
    )
      return [];
    const pullRequest = pullRequestsByTaskID.get(task.id);
    if (
      !query.terms.every((term) =>
        matchesText(task, feature, pullRequest, term),
      )
    )
      return [];
    return [pullRequest ? { task, feature, pullRequest } : { task, feature }];
  });
}

function matchesQualifiers(
  task: Task,
  pullRequest: PullRequest | undefined,
  qualifiers: TaskSearchQualifier[],
): boolean {
  return qualifiers.every((qualifier) => {
    if (qualifier.type === "task-status") {
      if (qualifier.value === "ready") return task.ready;
      return taskDisplayStateToken(task.displayState) === qualifier.value;
    }
    if (qualifier.type === "block")
      return task.blockLabels.some(
        (label) => blockQualifier(label) === qualifier.value,
      );
    if (qualifier.value === "error") return Boolean(pullRequest?.syncError);
    return (
      pullRequest !== undefined &&
      pullRequestDisplayStateToken(pullRequest.displayState) === qualifier.value
    );
  });
}

function matchesText(
  task: Task,
  feature: Feature,
  pullRequest: PullRequest | undefined,
  term: string,
): boolean {
  const values = [
    task.id,
    task.title,
    task.scope,
    task.assignee,
    feature.id,
    feature.title,
    pullRequest?.host,
    pullRequest?.owner,
    pullRequest?.repository,
    pullRequest?.author,
  ];
  const needle = normalizeText(term);
  return values.some(
    (value) => value !== undefined && normalizeText(value).includes(needle),
  );
}

function normalizeText(value: string): string {
  return value.normalize("NFKC").toLowerCase();
}

function isTaskStatusValue(value: string): value is TaskStatusValue {
  return (taskStatusValues as readonly string[]).includes(value);
}

function isGithubStatusValue(value: string): value is GithubStatusValue {
  return (githubStatusValues as readonly string[]).includes(value);
}

function isBlockValue(value: string): value is BlockValue {
  return (blockValues as readonly string[]).includes(value);
}

// 検索語は表示ラベルより短くする。依存未解決は 1 種類しかないので dependency で
// 足りる。
function blockQualifier(label: TaskBlockLabel): BlockValue | "" {
  if (label === TaskBlockLabel.DEPENDENCY_UNRESOLVED) return "dependency";
  if (label === TaskBlockLabel.CONFLICT) return "conflict";
  if (label === TaskBlockLabel.CHANGES_REQUESTED) return "changes-requested";
  if (label === TaskBlockLabel.CI_FAILED) return "ci-failed";
  return "";
}

function tokenize(input: string): {
  tokens: SearchToken[];
  error?: TaskSearchParseError;
} {
  const tokens: SearchToken[] = [];
  let value = "";
  let quoted = false;
  let quoteOpen = false;
  let escaped = false;
  let started = false;
  const push = () => {
    if (!started) return;
    tokens.push({ value, quoted });
    value = "";
    quoted = false;
    started = false;
  };

  for (const character of input) {
    if (escaped) {
      value += character;
      escaped = false;
      started = true;
      continue;
    }
    if (character === "\\" && quoteOpen) {
      escaped = true;
      started = true;
      continue;
    }
    if (character === '"') {
      quoteOpen = !quoteOpen;
      quoted = true;
      started = true;
      continue;
    }
    if (/\s/u.test(character) && !quoteOpen) {
      push();
      continue;
    }
    value += character;
    started = true;
  }
  if (quoteOpen) return { tokens: [], error: { type: "unterminated-quote" } };
  push();
  return { tokens };
}
