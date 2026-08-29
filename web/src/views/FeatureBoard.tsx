import { Link } from "@tanstack/react-router";
import { useTranslation } from "react-i18next";
import type { Feature } from "../gen/prx/v1/prx_pb";

interface FeatureBoardProps {
  features: Feature[];
}

export function FeatureBoard({ features }: FeatureBoardProps) {
  const { t } = useTranslation();
  return (
    <section className="feature-board">
      <header>
        <p className="section-label">{t("dashboard.featureTelemetry")}</p>
        <h2>{t("dashboard.deliveryLines")}</h2>
      </header>
      {features.length === 0 ? (
        <div className="empty compact">
          <h3>{t("dashboard.noFeaturesTitle")}</h3>
          <p>{t("dashboard.noFeaturesDetail")}</p>
        </div>
      ) : (
        <div className="feature-table">
          {features.map((feature) => (
            <Link
              key={feature.id}
              to="/features/$featureId"
              params={{ featureId: feature.id }}
              className="feature-row"
            >
              <div>
                <b>{feature.title}</b>
                <small>{feature.slug}</small>
              </div>
              <div className="progress-track">
                <i
                  style={{
                    width: `${feature.taskCount ? (feature.mergedCount / feature.taskCount) * 100 : 0}%`,
                  }}
                />
              </div>
              <span>
                {feature.mergedCount}/{feature.taskCount}
              </span>
              <strong
                className={
                  feature.readyCount
                    ? "status-ready"
                    : feature.conflictCount
                      ? "status-conflict"
                      : "status-none"
                }
              >
                {feature.readyCount
                  ? t("dashboard.featureReady", {
                      count: feature.readyCount,
                    })
                  : feature.conflictCount
                    ? t("dashboard.featureBlocked", {
                        count: feature.conflictCount,
                      })
                    : t("dashboard.steady")}
              </strong>
            </Link>
          ))}
        </div>
      )}
    </section>
  );
}
