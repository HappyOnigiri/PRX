import { describe, expect, it } from "vitest";
import {
  selectFeatureTab,
  validateFeatureTabSearch,
} from "../src/feature-tabs";
import { FeatureStatus } from "../src/gen/prx/v1/prx_pb";
import { makeFeature } from "./factories";

const features = [
  makeFeature({ id: "F-1", title: "Checkout" }),
  makeFeature({
    id: "F-2",
    title: "Indexing",
    displayStatus: FeatureStatus.COMPLETED,
  }),
  makeFeature({ id: "F-3", title: "Legacy", archived: true }),
];

describe("feature tabs", () => {
  it("selects the features each tab stands for", () => {
    expect(selectFeatureTab(features, "active").map((f) => f.id)).toEqual([
      "F-1",
    ]);
    expect(selectFeatureTab(features, "completed").map((f) => f.id)).toEqual([
      "F-2",
    ]);
    expect(selectFeatureTab(features, "archived").map((f) => f.id)).toEqual([
      "F-3",
    ]);
  });

  // 手で書き換えた古いリンクでも、作業対象の集合でページは開く。
  it("falls back to the active tab for an unknown or missing value", () => {
    expect(validateFeatureTabSearch({ features: "archived" })).toEqual({
      features: "archived",
    });
    expect(validateFeatureTabSearch({ features: "everything" })).toEqual({
      features: "active",
    });
    expect(validateFeatureTabSearch({})).toEqual({ features: "active" });
  });
});
