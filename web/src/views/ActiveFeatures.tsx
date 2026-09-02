import { isActiveFeature } from "../feature-status";
import { FeatureListPage } from "./FeatureListPage";

export function ActiveFeatures() {
  return <FeatureListPage namespace="active" select={isActiveFeature} />;
}
