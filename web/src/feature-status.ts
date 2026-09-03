import { FeatureStatus, type Feature } from "./gen/prx/v1/prx_pb";

// The overview, the sidebar, and the task queues all present work in flight.
// That excludes read-only features and the ones the server presents as
// completed, each of which has its own page.
//
// Every category reads the server's readOnly rather than the feature's own
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

// Every feature belongs to exactly one category. The identifier doubles as the
// translation namespace of the category's list page, so the sidebar, the route
// table, and the pages all read the same wording from one definition.
export const featureCategoryIds = ["active", "completed", "archived"] as const;
export type FeatureCategoryId = (typeof featureCategoryIds)[number];

export interface FeatureCategory {
  id: FeatureCategoryId;
  path: `/${FeatureCategoryId}`;
  navLabelKey: `nav.${FeatureCategoryId}Features`;
  select: (feature: Feature) => boolean;
}

const categoriesById: Record<FeatureCategoryId, FeatureCategory> = {
  active: {
    id: "active",
    path: "/active",
    navLabelKey: "nav.activeFeatures",
    select: isActiveFeature,
  },
  completed: {
    id: "completed",
    path: "/completed",
    navLabelKey: "nav.completedFeatures",
    select: isCompletedFeature,
  },
  archived: {
    id: "archived",
    path: "/archived",
    navLabelKey: "nav.archivedFeatures",
    select: isArchivedFeature,
  },
};

export const featureCategories: readonly FeatureCategory[] =
  featureCategoryIds.map((id) => categoriesById[id]);

export function featureCategoryById(id: FeatureCategoryId): FeatureCategory {
  return categoriesById[id];
}

// featureCategoryForPath reports the category a route presents, and nothing
// for the routes that present something else. The sidebar keeps its own
// selection on those, so the overview and task search leave it untouched.
export function featureCategoryForPath(
  pathname: string,
): FeatureCategory | undefined {
  const path = pathname.length > 1 ? pathname.replace(/\/+$/, "") : pathname;
  return featureCategories.find((entry) => entry.path === path);
}
