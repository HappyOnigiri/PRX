import {
  isActiveFeature,
  isArchivedFeature,
  isCompletedFeature,
} from "./feature-status";
import type { Feature } from "./gen/prx/v1/prx_pb";

// A feature list is filtered by status on the project page it appears on. The
// tab is a search parameter rather than browser-local state so reload, history,
// and a shared link reproduce the view, as task search does with its query.
export const featureTabIds = ["active", "completed", "archived"] as const;
export type FeatureTabId = (typeof featureTabIds)[number];

const predicates: Record<FeatureTabId, (feature: Feature) => boolean> = {
  active: isActiveFeature,
  completed: isCompletedFeature,
  archived: isArchivedFeature,
};

export function selectFeatureTab(
  features: Feature[],
  tab: FeatureTabId,
): Feature[] {
  return features.filter(predicates[tab]);
}

function isFeatureTabId(value: unknown): value is FeatureTabId {
  return featureTabIds.includes(value as FeatureTabId);
}

// An unknown or missing value falls back to the working set rather than
// failing the route: a hand-edited or stale link still opens the page.
export function validateFeatureTabSearch(search: Record<string, unknown>): {
  features: FeatureTabId;
} {
  const value = search["features"];
  return { features: isFeatureTabId(value) ? value : "active" };
}
