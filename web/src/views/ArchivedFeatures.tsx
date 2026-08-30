import { Link } from "@tanstack/react-router";
import { useTranslation } from "react-i18next";
import { useSnapshot } from "../hooks";
import { featureStatusLabel, formatError } from "../i18n/domain";
import { StateMessage } from "./Dashboard";

export function ArchivedFeatures() {
  const { t } = useTranslation();
  const { data, isPending, error, refetch } = useSnapshot();

  if (isPending)
    return (
      <StateMessage
        title={t("archived.loadingTitle")}
        detail={t("archived.loadingDetail")}
      />
    );
  if (error)
    return (
      <StateMessage
        title={t("archived.errorTitle")}
        detail={formatError(error, t)}
        action={() => void refetch()}
      />
    );

  const features = data.features.filter((feature) => feature.archived);
  return (
    <div className="dashboard archived-page">
      <header className="page-head">
        <div>
          <p className="section-label">{t("archived.eyebrow")}</p>
          <h1>{t("archived.title")}</h1>
          <p>{t("archived.description")}</p>
        </div>
        <div className="clock">
          <span>{features.length}</span>
          <small>{t("archived.featureCount")}</small>
        </div>
      </header>
      <section className="archived-list" aria-label={t("archived.listLabel")}>
        {features.length === 0 ? (
          <div className="empty compact">
            <h2>{t("archived.emptyTitle")}</h2>
            <p>{t("archived.emptyDetail")}</p>
          </div>
        ) : (
          features.map((feature) => (
            <Link
              key={feature.id}
              to="/features/$featureId"
              params={{ featureId: feature.id }}
              className="archived-row"
            >
              <div className="archived-row-title">
                <b>{feature.title}</b>
                <small>{feature.slug}</small>
              </div>
              <div className="progress-track" aria-hidden="true">
                <i
                  style={{
                    width: `${feature.taskCount ? (feature.mergedCount / feature.taskCount) * 100 : 0}%`,
                  }}
                />
              </div>
              <span>
                {t("archived.progress", {
                  merged: feature.mergedCount,
                  total: feature.taskCount,
                })}
              </span>
              <strong>{featureStatusLabel(feature.status, t)}</strong>
            </Link>
          ))
        )}
      </section>
    </div>
  );
}
