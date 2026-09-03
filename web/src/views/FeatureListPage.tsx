import { useTranslation } from "react-i18next";
import type { FeatureCategoryId } from "../feature-status";
import type { Feature } from "../gen/prx/v1/prx_pb";
import { useSnapshot } from "../hooks";
import { formatError } from "../i18n/domain";
import { StateMessage } from "./Dashboard";
import { FeatureListRow } from "./FeatureListRow";

// Each feature category presents the same rows with the same progress and
// status, so they share one page and differ only in which features they select
// and which wording they read.
interface FeatureListPageProps {
  namespace: FeatureCategoryId;
  select: (feature: Feature) => boolean;
}

export function FeatureListPage({ namespace, select }: FeatureListPageProps) {
  const { t } = useTranslation();
  const { data, isPending, error, refetch } = useSnapshot();

  if (isPending)
    return (
      <StateMessage
        title={t(`${namespace}.loadingTitle`)}
        detail={t(`${namespace}.loadingDetail`)}
      />
    );
  if (error)
    return (
      <StateMessage
        title={t(`${namespace}.errorTitle`)}
        detail={formatError(error, t)}
        action={() => void refetch()}
      />
    );

  const features = data.features.filter(select);
  const projectTitles = new Map(
    data.projects.map((project) => [project.id, project.title]),
  );
  return (
    <div className="dashboard feature-list-page">
      <header className="page-head">
        <div>
          <p className="section-label">{t(`${namespace}.eyebrow`)}</p>
          <h1>{t(`${namespace}.title`)}</h1>
          <p>{t(`${namespace}.description`)}</p>
        </div>
        <div className="clock">
          <span>{features.length}</span>
          <small>{t(`${namespace}.featureCount`)}</small>
        </div>
      </header>
      <section
        className="feature-list"
        aria-label={t(`${namespace}.listLabel`)}
      >
        {features.length === 0 ? (
          <div className="empty compact">
            <h2>{t(`${namespace}.emptyTitle`)}</h2>
            <p>{t(`${namespace}.emptyDetail`)}</p>
          </div>
        ) : (
          features.map((feature) => (
            <FeatureListRow
              key={feature.id}
              feature={feature}
              progressLabel={t(`${namespace}.progress`, {
                merged: feature.mergedCount,
                total: feature.taskCount,
              })}
              projectTitle={projectTitles.get(feature.projectId)}
            />
          ))
        )}
      </section>
    </div>
  );
}
