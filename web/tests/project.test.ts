import { describe, expect, it } from "vitest";
import {
  documentsInProject,
  featuresInProject,
  projectsByArchive,
} from "../src/project";
import { makeDocument, makeFeature, makeProject } from "./factories";

describe("project selectors", () => {
  it("splits projects by their archived flag", () => {
    const projects = [
      makeProject({ id: "P-1", title: "Active" }),
      makeProject({ id: "P-2", title: "Retired", archived: true }),
    ];
    expect(projectsByArchive(projects, false).map((item) => item.id)).toEqual([
      "P-1",
    ]);
    expect(projectsByArchive(projects, true).map((item) => item.id)).toEqual([
      "P-2",
    ]);
  });

  // An unaffiliated feature or document carries an empty project ID, so a
  // caller holding one must not collect every unaffiliated record.
  it("never matches unaffiliated records against an empty project ID", () => {
    const features = [
      makeFeature({ id: "F-1", projectId: "P-1" }),
      makeFeature({ id: "F-2" }),
    ];
    const documents = [
      makeDocument({ id: "doc-1", taskId: "", projectId: "P-1" }),
      makeDocument({ id: "doc-2" }),
    ];
    expect(featuresInProject(features, "P-1").map((item) => item.id)).toEqual([
      "F-1",
    ]);
    expect(featuresInProject(features, "")).toEqual([]);
    expect(documentsInProject(documents, "P-1").map((item) => item.id)).toEqual(
      ["doc-1"],
    );
    expect(documentsInProject(documents, "")).toEqual([]);
  });
});
