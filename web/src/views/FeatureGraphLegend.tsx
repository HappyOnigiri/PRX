import { useTranslation } from "react-i18next";

interface FeatureGraphLegendProps {
  taskCount: number;
  dependencyCount: number;
}

export function FeatureGraphLegend({
  taskCount,
  dependencyCount,
}: FeatureGraphLegendProps) {
  const { t } = useTranslation();
  return (
    <div className="graph-legend">
      <span>
        <i className="ready" />
        {t("workspace.legend.ready")}
      </span>
      <span>
        <i className="review" />
        {t("workspace.legend.review")}
      </span>
      <span>
        <i className="conflict" />
        {t("workspace.legend.conflict")}
      </span>
      <span>
        <i className="merged" />
        {t("workspace.legend.merged")}
      </span>
      <b>
        {t("workspace.graphSummary", {
          nodes: taskCount,
          links: dependencyCount,
        })}
      </b>
    </div>
  );
}
