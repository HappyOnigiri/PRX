import { Link } from "@tanstack/react-router";
import { useTranslation } from "react-i18next";
import type { Feature } from "../gen/prx/v1/prx_pb";
import { featureStatusLabel } from "../i18n/domain";
import { EntityIcon } from "./EntityIcon";

interface FeatureListRowProps {
  feature: Feature;
  // The progress wording belongs to the page the row appears on, so the caller
  // resolves it and the row stays free of a fixed translation namespace.
  progressLabel: string;
}

export function FeatureListRow({
  feature,
  progressLabel,
}: FeatureListRowProps) {
  const { t } = useTranslation();
  return (
    <Link
      to="/features/$featureId"
      params={{ featureId: feature.id }}
      className="feature-list-row"
    >
      <div className="feature-list-row-title">
        <b>
          <EntityIcon kind="feature" size={15} />
          {feature.title}
        </b>
        <small>{feature.slug}</small>
      </div>
      <div className="progress-track" aria-hidden="true">
        <i
          style={{
            width: `${feature.taskCount ? (feature.finishedCount / feature.taskCount) * 100 : 0}%`,
          }}
        />
      </div>
      <span>{progressLabel}</span>
      <strong>{featureStatusLabel(feature.displayStatus, t)}</strong>
    </Link>
  );
}
