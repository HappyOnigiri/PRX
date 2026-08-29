import { Link } from "@tanstack/react-router";
import { useTranslation } from "react-i18next";
import type { Feature } from "../gen/prx/v1/prx_pb";

interface FeatureNavigationProps {
  features: Feature[] | undefined;
}

export function FeatureNavigation({ features }: FeatureNavigationProps) {
  const { t } = useTranslation();
  return (
    <nav aria-label={t("nav.features")}>
      <Link to="/" className="nav-link">
        {t("nav.overview")} <span>{features?.length ?? "—"}</span>
      </Link>
      <div className="nav-caption">{t("nav.activeCircuits")}</div>
      {features
        ?.filter((feature) => !feature.archived)
        .map((feature) => (
          <Link
            key={feature.id}
            to="/features/$featureId"
            params={{ featureId: feature.id }}
            className="feature-link"
            activeProps={{ "data-active": true }}
          >
            <i
              className={
                feature.conflictCount
                  ? "pulse conflict"
                  : feature.readyCount
                    ? "pulse ready"
                    : "pulse"
              }
            />
            <span>{feature.title}</span>
            <b>
              {feature.mergedCount}/{feature.taskCount}
            </b>
          </Link>
        ))}
    </nav>
  );
}
