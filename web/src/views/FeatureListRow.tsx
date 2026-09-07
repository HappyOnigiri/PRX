import { Link } from "@tanstack/react-router";
import { useTranslation } from "react-i18next";
import type { Feature } from "../gen/prx/v1/prx_pb";
import { featureStatusLabel, featureStatusToken } from "../i18n/domain";
import { EntityIcon } from "./EntityIcon";
import { StatusBadge } from "./StatusBadge";

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
      {/* The state opens the line everywhere a titled row appears, so a reader
          scans one column of states down the list. */}
      <div className="feature-list-row-title">
        <StatusBadge
          className={`status-${featureStatusToken(feature.displayStatus)}`}
          label={featureStatusLabel(feature.displayStatus, t)}
        />
        <b>
          <EntityIcon kind="feature" size={15} />
          {feature.title}
        </b>
      </div>
      <div className="progress-track" aria-hidden="true">
        <i
          style={{
            width: `${feature.taskCount ? (feature.finishedCount / feature.taskCount) * 100 : 0}%`,
          }}
        />
      </div>
      <span>{progressLabel}</span>
    </Link>
  );
}
