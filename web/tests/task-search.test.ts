import { describe, expect, it } from "vitest";
import {
  PullRequestDisplayState,
  TaskDisplayState,
} from "../src/gen/prx/v1/prx_pb";
import { filterTaskSearchResults, parseTaskSearch } from "../src/task-search";
import {
  makeFeature,
  makePullRequest,
  makeSnapshot,
  makeTask,
} from "./factories";

describe("task search", () => {
  it("parses status qualifiers, quoted terms, and plain key-value words", () => {
    expect(
      parseTaskSearch("task-status:READY github-status:error payments"),
    ).toEqual({
      qualifiers: [
        { type: "task-status", value: "ready" },
        { type: "github-status", value: "error" },
      ],
      terms: ["payments"],
    });
    expect(parseTaskSearch('"task-status:ready" foo:bar')).toEqual({
      qualifiers: [],
      terms: ["task-status:ready", "foo:bar"],
    });
  });

  it("reports invalid recognized values and unclosed quotes", () => {
    expect(parseTaskSearch("task-status:waiting").error).toEqual({
      type: "invalid-qualifier",
      key: "task-status",
      value: "waiting",
    });
    expect(parseTaskSearch('payments "review').error).toEqual({
      type: "unterminated-quote",
    });
  });

  it("filters active tasks with AND qualifiers and Unicode-insensitive terms", () => {
    const ready = makeTask({
      id: "task-ready",
      title: "決済 API",
      assignee: "Alice",
      ready: true,
      displayState: TaskDisplayState.NOT_STARTED,
    });
    const review = makeTask({
      id: "task-review",
      title: "Review checkout",
      ready: false,
      displayState: TaskDisplayState.REVIEW_WAITING,
    });
    const archived = makeTask({
      id: "task-archived",
      featureId: "feature-archived",
      title: "決済 API",
    });
    const active = makeFeature({ id: "feature-1", title: "決済 rollout" });
    const archivedFeature = makeFeature({
      id: "feature-archived",
      archived: true,
      title: "Archived",
    });
    const query = parseTaskSearch("task-status:ready github-status:open ALICE");
    if (query.error) throw new Error("test query did not parse");
    const results = filterTaskSearchResults(
      makeSnapshot({
        features: [active, archivedFeature],
        tasks: [ready, review, archived],
        pullRequests: [
          makePullRequest({
            taskId: ready.id,
            host: "github.example",
            author: "alice",
            displayState: PullRequestDisplayState.OPEN,
          }),
          makePullRequest({
            taskId: review.id,
            displayState: PullRequestDisplayState.REVIEW_WAITING,
            stale: true,
          }),
        ],
      }),
      query,
    );
    expect(results.map(({ task }) => task.id)).toEqual([ready.id]);

    const JapaneseResults = filterTaskSearchResults(
      makeSnapshot({
        features: [active, archivedFeature],
        tasks: [ready, review, archived],
      }),
      parseTaskSearch('"決済 API"'),
    );
    expect(JapaneseResults.map(({ task }) => task.id)).toEqual([ready.id]);
  });

  it("treats sync errors separately from stale terminal data", () => {
    const stale = makeTask({ id: "stale", title: "Stale only" });
    const failed = makeTask({ id: "failed", title: "Failed sync" });
    const snapshot = makeSnapshot({
      tasks: [stale, failed],
      pullRequests: [
        makePullRequest({ taskId: stale.id, stale: true }),
        makePullRequest({
          taskId: failed.id,
          stale: true,
          syncError: "GitHub unavailable",
          displayState: PullRequestDisplayState.MERGED,
        }),
      ],
    });
    const results = filterTaskSearchResults(
      snapshot,
      parseTaskSearch("github-status:error"),
    );
    expect(results.map(({ task }) => task.id)).toEqual([failed.id]);
  });
});
