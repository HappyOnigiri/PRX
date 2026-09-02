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

// Every feature belongs to exactly one category. The identifier doubles as the
// translation namespace of the category's list page, so the sidebar, the route
// table, and the pages all read the same wording from one definition.
export type FeatureCategoryId = "active" | "completed" | "archived";

export interface FeatureCategory {
  id: FeatureCategoryId;
  path: `/${FeatureCategoryId}`;
  navLabelKey: `nav.${FeatureCategoryId}Features`;
  select: (feature: Feature) => boolean;
}

const activeCategory: FeatureCategory = {
  id: "active",
  path: "/active",
  navLabelKey: "nav.activeFeatures",
  select: isActiveFeature,
};
const completedCategory: FeatureCategory = {
  id: "completed",
  path: "/completed",
  navLabelKey: "nav.completedFeatures",
  select: isCompletedFeature,
};
const archivedCategory: FeatureCategory = {
  id: "archived",
  path: "/archived",
  navLabelKey: "nav.archivedFeatures",
  select: isArchivedFeature,
};

export const featureCategories: readonly FeatureCategory[] = [
  activeCategory,
  completedCategory,
  archivedCategory,
];

// Archiving wins over completion because an archived feature leaves the
// working set entirely, which is the same order the list pages select in.
function featureCategoryOf(feature: Feature): FeatureCategory {
  if (isArchivedFeature(feature)) return archivedCategory;
  if (isCompletedFeature(feature)) return completedCategory;
  return activeCategory;
}

const featureDetailPrefix = "/features/";

// selectedFeatureCategory derives the sidebar selection from the current URL
// so that reload, history, and sharing reproduce it without stored state. A
// feature workspace selects the category its feature belongs to; anything else
// falls back to the working set, which is also what an unloaded snapshot shows.
export function selectedFeatureCategory(
  pathname: string,
  features: readonly Feature[] | undefined,
): FeatureCategory {
  const path = pathname.length > 1 ? pathname.replace(/\/+$/, "") : pathname;
  const category = featureCategories.find((entry) => entry.path === path);
  if (category) return category;
  if (path.startsWith(featureDetailPrefix)) {
    const featureId = decodeURIComponent(
      path.slice(featureDetailPrefix.length),
    );
    const feature = features?.find((entry) => entry.id === featureId);
    if (feature) return featureCategoryOf(feature);
  }
  return activeCategory;
}
