import { describe, expect, it } from "vitest";
import { TaskBlockLabel, TaskDisplayState } from "../src/gen/prx/v1/prx_pb";
import { isDependencyBlockedTask, isDormantTask } from "../src/task-attention";

describe("task attention", () => {
  it("sinks settled tasks without treating them as blocker-bound", () => {
    for (const displayState of [
      TaskDisplayState.COMPLETED,
      TaskDisplayState.CLOSED,
      TaskDisplayState.MERGED,
    ]) {
      const task = {
        displayState,
        blockLabels: [TaskBlockLabel.DEPENDENCY_UNRESOLVED],
      };
      expect(isDormantTask(task)).toBe(true);
      // 決着した task には作業が残っていないので、破線の枠は残さない。
      expect(isDependencyBlockedTask(task)).toBe(false);
    }
  });

  it("sinks a task waiting on a blocker and keeps it marked as blocked", () => {
    const task = {
      displayState: TaskDisplayState.IMPLEMENTED,
      blockLabels: [TaskBlockLabel.DEPENDENCY_UNRESOLVED],
    };
    expect(isDormantTask(task)).toBe(true);
    expect(isDependencyBlockedTask(task)).toBe(true);
  });

  it("leaves a task raised when its own pull request is what needs work", () => {
    // コンフリクトも修正依頼もその task で手を動かせるので、面は沈めない。
    const task = {
      displayState: TaskDisplayState.IMPLEMENTED,
      blockLabels: [TaskBlockLabel.CONFLICT, TaskBlockLabel.CHANGES_REQUESTED],
    };
    expect(isDormantTask(task)).toBe(false);
    expect(isDependencyBlockedTask(task)).toBe(false);
  });
});
