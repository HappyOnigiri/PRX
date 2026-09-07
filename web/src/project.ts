import type { Document, Feature, Project } from "./gen/prx/v1/prx_pb";

export function projectsByArchive(
  projects: Project[],
  archived: boolean,
): Project[] {
  return projects.filter((project) => project.archived === archived);
}

export function featuresInProject(
  features: Feature[],
  projectId: string,
): Feature[] {
  return features.filter((feature) => feature.projectId === projectId);
}

export function documentsInProject(
  documents: Document[],
  projectId: string,
): Document[] {
  return documents.filter(
    (document) => document.projectId !== "" && document.projectId === projectId,
  );
}
