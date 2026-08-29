import { describe, expect, it } from "vitest";
import {
  BlockedReasonCode,
  DocumentKind,
  FeatureStatus,
  PullRequestDisplayState,
  TaskKind,
  TaskStatus,
  type BlockedReason,
} from "../src/gen/prx/v1/prx_pb";
import i18n from "../src/i18n";
import {
  blockedReasonLabel,
  documentKindLabel,
  featureStatusLabel,
  pullRequestDisplayStateLabel,
  taskKindLabel,
  taskStatusLabel,
} from "../src/i18n/domain";

function numericValues(value: object): number[] {
  return Object.values(value).filter(
    (entry): entry is number => typeof entry === "number",
  );
}

describe("domain labels", () => {
  const t = i18n.getFixedT("en");

  it("localizes all feature, task, and document enum values", () => {
    for (const value of numericValues(FeatureStatus))
      expect(featureStatusLabel(value, t)).toEqual(expect.any(String));
    for (const value of numericValues(TaskStatus))
      expect(taskStatusLabel(value, t)).toEqual(expect.any(String));
    for (const value of numericValues(TaskKind))
      expect(taskKindLabel(value, t)).toEqual(expect.any(String));
    for (const value of numericValues(DocumentKind))
      expect(documentKindLabel(value, t)).toEqual(expect.any(String));
    for (const value of numericValues(PullRequestDisplayState))
      expect(pullRequestDisplayStateLabel(value, t)).toEqual(
        expect.any(String),
      );
  });

  it("explains each dependency blocker and handles missing reasons", () => {
    const title = (id: string) => (id === "task-1" ? "Build API" : undefined);
    expect(blockedReasonLabel(undefined, title, t)).toBe("");
    expect(
      blockedReasonLabel(
        {
          code: BlockedReasonCode.BLOCKER_STALE,
          blockerTaskId: "task-1",
        } as BlockedReason,
        title,
        t,
      ),
    ).toContain("Build API");
    expect(
      blockedReasonLabel(
        {
          code: BlockedReasonCode.WAITING_FOR_BLOCKER,
          blockerTaskId: "unknown",
        } as BlockedReason,
        title,
        t,
      ),
    ).toContain("an upstream task");
    expect(
      blockedReasonLabel(
        {
          code: BlockedReasonCode.DEPENDENCY_DATA_INCOMPLETE,
          blockerTaskId: "task-1",
        } as BlockedReason,
        title,
        t,
      ),
    ).toBe("Dependency data is incomplete");
  });
});
