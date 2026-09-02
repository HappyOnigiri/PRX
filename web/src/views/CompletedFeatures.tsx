import { isCompletedFeature } from "../feature-status";
import { FeatureListPage } from "./FeatureListPage";

export function CompletedFeatures() {
  return <FeatureListPage namespace="completed" select={isCompletedFeature} />;
}
