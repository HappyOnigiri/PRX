import { describe, expect, it } from "vitest";
import {
  featureCategories,
  selectedFeatureCategory,
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
});

describe("selectedFeatureCategory", () => {
  it("selects the category a list route presents", () => {
    expect(selectedFeatureCategory("/active", features).id).toBe("active");
    expect(selectedFeatureCategory("/completed", features).id).toBe(
      "completed",
    );
    expect(selectedFeatureCategory("/archived", features).id).toBe("archived");
    expect(selectedFeatureCategory("/archived/", features).id).toBe("archived");
  });

  it("selects the category the open feature belongs to", () => {
    expect(selectedFeatureCategory("/features/active", features).id).toBe(
      "active",
    );
    expect(selectedFeatureCategory("/features/completed", features).id).toBe(
      "completed",
    );
    expect(selectedFeatureCategory("/features/archived", features).id).toBe(
      "archived",
    );
    expect(
      selectedFeatureCategory("/features/archived%20and%20completed", features)
        .id,
    ).toBe("archived");
  });

  it("falls back to the working set outside the category routes", () => {
    expect(selectedFeatureCategory("/", features).id).toBe("active");
    expect(selectedFeatureCategory("/tasks", features).id).toBe("active");
    expect(selectedFeatureCategory("/nowhere", features).id).toBe("active");
    expect(selectedFeatureCategory("/features/unknown", features).id).toBe(
      "active",
    );
    expect(selectedFeatureCategory("/features/archived", undefined).id).toBe(
      "active",
    );
  });
});
