import { useTranslation } from "react-i18next";
import type { Feature } from "../gen/prx/v1/prx_pb";
import { FeatureListRow } from "./FeatureListRow";

// The project already names itself, so the rows omit the project label the
// category list pages show.
export function ProjectFeatureList({ features }: { features: Feature[] }) {
  const { t } = useTranslation();
  return (
    <section className="feature-list" aria-label={t("project.featuresLabel")}>
      <header className="project-section-head">
        <h2>{t("project.featuresTitle")}</h2>
        <span>{t("project.featureCount", { count: features.length })}</span>
      </header>
      {features.length === 0 ? (
        <div className="empty compact">
          <h3>{t("project.noFeaturesTitle")}</h3>
          <p>{t("project.noFeaturesDetail")}</p>
        </div>
      ) : (
        features.map((feature) => (
          <FeatureListRow
            key={feature.id}
            feature={feature}
            progressLabel={t("project.progress", {
              merged: feature.mergedCount,
              total: feature.taskCount,
            })}
          />
        ))
      )}
    </section>
  );
}
