import type { Document, Feature, Project } from "./gen/prx/v1/prx_pb";

export function projectsByArchive(
  projects: Project[],
  archived: boolean,
): Project[] {
  return projects.filter((project) => project.archived === archived);
}

// An unaffiliated feature carries an empty projectId rather than an absent one,
// so the comparison has to reject the empty value; otherwise a caller holding
// an empty project ID would collect every unaffiliated feature.
export function featuresInProject(
  features: Feature[],
  projectId: string,
): Feature[] {
  return features.filter(
    (feature) => feature.projectId !== "" && feature.projectId === projectId,
  );
}

// The unaffiliated features get their own row in the sidebar and their own
// page, so they need a selector as much as a project's members do.
export function featuresWithoutProject(features: Feature[]): Feature[] {
  return features.filter((feature) => feature.projectId === "");
}

// The unaffiliated row is not a project, so it has no project ID to key its
// expansion state by. A project ID is always "P-<number>", so this literal
// cannot collide with one.
export const unassignedProjectKey = "unassigned";

export function documentsInProject(
  documents: Document[],
  projectId: string,
): Document[] {
  return documents.filter(
    (document) => document.projectId !== "" && document.projectId === projectId,
  );
}
