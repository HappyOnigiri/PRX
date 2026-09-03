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

export function documentsInProject(
  documents: Document[],
  projectId: string,
): Document[] {
  return documents.filter(
    (document) => document.projectId !== "" && document.projectId === projectId,
  );
}
