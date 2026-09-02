import { describe, expect, it } from "vitest";
import {
  featureCategories,
  featureCategoryById,
  featureCategoryForPath,
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

describe("featureCategories", () => {
  it("pairs every category with its own route and selector", () => {
    expect(featureCategories.map((category) => category.id)).toEqual([
      "active",
      "completed",
      "archived",
    ]);
    expect(featureCategories.map((category) => category.path)).toEqual([
      "/active",
      "/completed",
      "/archived",
    ]);
    expect(
      featureCategories.map((category) =>
        features.filter(category.select).map((feature) => feature.id),
      ),
    ).toEqual([
      ["active"],
      ["completed"],
      ["archived", "archived and completed"],
    ]);
  });

  it("looks a category up by the identifier that was stored", () => {
    expect(featureCategoryById("completed").path).toBe("/completed");
    expect(featureCategoryById("archived").navLabelKey).toBe(
      "nav.archivedFeatures",
    );
  });
});

describe("featureCategoryForPath", () => {
  it("reports the category a list route presents", () => {
    expect(featureCategoryForPath("/active")?.id).toBe("active");
    expect(featureCategoryForPath("/completed")?.id).toBe("completed");
    expect(featureCategoryForPath("/archived")?.id).toBe("archived");
    expect(featureCategoryForPath("/archived/")?.id).toBe("archived");
  });

  it("reports nothing for the routes that present something else", () => {
    expect(featureCategoryForPath("/")).toBeUndefined();
    expect(featureCategoryForPath("/tasks")).toBeUndefined();
    expect(featureCategoryForPath("/features/completed")).toBeUndefined();
    expect(featureCategoryForPath("/nowhere")).toBeUndefined();
  });
});
