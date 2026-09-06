import { describe, expect, it } from "vitest";
import {
  isActiveFeature,
  isArchivedFeature,
  isCompletedFeature,
  unfinishedTaskCount,
} from "../src/feature-status";
import { FeatureStatus } from "../src/gen/prx/v1/prx_pb";
import { makeFeature } from "./factories";

const features = [
  makeFeature({ id: "active", title: "Active feature" }),
  makeFeature({
    id: "completed",
    title: "Completed feature",
    displayStatus: FeatureStatus.COMPLETED,
  }),
  makeFeature({ id: "archived", title: "Archived feature", archived: true }),
  makeFeature({
    id: "archived and completed",
    title: "Archived completed feature",
    archived: true,
    displayStatus: FeatureStatus.COMPLETED,
  }),
];

function selected(predicate: (feature: (typeof features)[number]) => boolean) {
  return features.filter(predicate).map((feature) => feature.id);
}

describe("feature status predicates", () => {
  // A feature that is both read-only and completed belongs to the archive, so
  // the three predicates have to partition the set rather than overlap.
  it("puts every feature in exactly one of the three states", () => {
    expect(selected(isActiveFeature)).toEqual(["active"]);
    expect(selected(isCompletedFeature)).toEqual(["completed"]);
    expect(selected(isArchivedFeature)).toEqual([
      "archived",
      "archived and completed",
    ]);
  });

  it("counts the tasks the completion rule still calls unfinished", () => {
    expect(
      unfinishedTaskCount(makeFeature({ taskCount: 5, finishedCount: 2 })),
    ).toBe(3);
  });
});
