import { FeatureStatus, type Feature } from "./gen/prx/v1/prx_pb";

// The overview, the sidebar, and the task queues all present work in flight.
// That excludes read-only features and the ones the server presents as
// completed, each of which has its own page.
//
// Every predicate reads the server's readOnly rather than the feature's own
// archived flag, because a feature inside an archived project is read-only too
// and belongs in the archive with the rest of it. The server derives that
// value; the browser must not recombine the two flags itself.
export function isActiveFeature(feature: Feature): boolean {
  return !feature.readOnly && feature.displayStatus !== FeatureStatus.COMPLETED;
}

// A feature that is both read-only and completed belongs to the archive, so the
// completed list only claims the ones still in the working set.
export function isCompletedFeature(feature: Feature): boolean {
  return !feature.readOnly && feature.displayStatus === FeatureStatus.COMPLETED;
}

export function isArchivedFeature(feature: Feature): boolean {
  return feature.readOnly;
}

// unfinishedTaskCount reports how many tasks the automatic completion rule
// still counts as unfinished, using the server's own count.
export function unfinishedTaskCount(feature: Feature): number {
  return feature.taskCount - feature.finishedCount;
}
