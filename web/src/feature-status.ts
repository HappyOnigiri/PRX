import { FeatureStatus, type Feature } from "./gen/prx/v1/prx_pb";

// The overview, the sidebar, and the task queues all present work in flight.
// That excludes archived features and the ones the server presents as
// completed, each of which has its own page.
export function isActiveFeature(feature: Feature): boolean {
  return !feature.archived && feature.displayStatus !== FeatureStatus.COMPLETED;
}

// A feature that is both archived and completed belongs to the archive, so the
// completed list only claims the ones still in the working set.
export function isCompletedFeature(feature: Feature): boolean {
  return !feature.archived && feature.displayStatus === FeatureStatus.COMPLETED;
}

export function isArchivedFeature(feature: Feature): boolean {
  return feature.archived;
}

// unfinishedTaskCount reports how many tasks the automatic completion rule
// still counts as unfinished, using the server's own count.
export function unfinishedTaskCount(feature: Feature): number {
  return feature.taskCount - feature.finishedCount;
}
