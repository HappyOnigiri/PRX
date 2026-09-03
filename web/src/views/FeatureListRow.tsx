import { Link } from "@tanstack/react-router";
import { useTranslation } from "react-i18next";
import type { Feature } from "../gen/prx/v1/prx_pb";
import { featureStatusLabel } from "../i18n/domain";

interface FeatureListRowProps {
  feature: Feature;
  // The progress wording belongs to the page the row appears on, so the caller
  // resolves it and the row stays free of a fixed translation namespace.
  progressLabel: string;
  // projectTitle is the owning project's name, shown only where it adds
  // information: a project's own page already states it.
  projectTitle?: string | undefined;
}

export function FeatureListRow({
  feature,
  progressLabel,
  projectTitle,
}: FeatureListRowProps) {
  const { t } = useTranslation();
  return (
    <Link
      to="/features/$featureId"
      params={{ featureId: feature.id }}
      className="feature-list-row"
    >
      <div className="feature-list-row-title">
        <b>{feature.title}</b>
        <small>
          {projectTitle ? `${feature.slug} · ${projectTitle}` : feature.slug}
        </small>
      </div>
      <div className="progress-track" aria-hidden="true">
        <i
          style={{
            width: `${feature.taskCount ? (feature.mergedCount / feature.taskCount) * 100 : 0}%`,
          }}
        />
      </div>
      <span>{progressLabel}</span>
      <strong>{featureStatusLabel(feature.displayStatus, t)}</strong>
    </Link>
  );
}
