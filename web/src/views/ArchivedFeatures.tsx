import { isArchivedFeature } from "../feature-status";
import { FeatureListPage } from "./FeatureListPage";

export function ArchivedFeatures() {
  return <FeatureListPage namespace="archived" select={isArchivedFeature} />;
}
