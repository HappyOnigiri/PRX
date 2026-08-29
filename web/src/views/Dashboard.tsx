import { Link } from "@tanstack/react-router";
import { useTranslation } from "react-i18next";
import { useSnapshot } from "../hooks";
import { formatError } from "../i18n/domain";

const queueNames = [
  [
    "readyTasks",
    "dashboard.queues.ready.title",
    "dashboard.queues.ready.detail",
  ],
  [
    "reviewWaitingTasks",
    "dashboard.queues.review.title",
    "dashboard.queues.review.detail",
  ],
  [
    "conflictTasks",
    "dashboard.queues.conflicts.title",
    "dashboard.queues.conflicts.detail",
  ],
  [
    "staleTasks",
    "dashboard.queues.stale.title",
    "dashboard.queues.stale.detail",
  ],
] as const;

export function Dashboard() {
  const { t } = useTranslation();
  const { data, isPending, error, refetch } = useSnapshot();
  if (isPending)
    return (
      <StateMessage
        title={t("dashboard.loadingTitle")}
        detail={t("dashboard.loadingDetail")}
      />
    );
  if (error || !data)
    return (
      <StateMessage
        title={t("dashboard.errorTitle")}
        detail={error ? formatError(error, t) : t("dashboard.noData")}
        action={() => void refetch()}
      />
    );
  return (
    <div className="dashboard">
      <header className="page-head">
        <div>
          <h1>
            {t("dashboard.titleStart")}
            <em>{t("dashboard.titleEmphasis")}</em>
          </h1>
          <p>{t("dashboard.description")}</p>
        </div>
        <div className="clock">
          <span>{data.tasks.length}</span>
          <small>{t("dashboard.nodesUnderControl")}</small>
        </div>
      </header>
      <section
        className="queue-strip"
        aria-label={t("dashboard.roadmapStatus")}
      >
        {queueNames.map(([key, title, detail]) => (
          <article key={key} className={`queue-meter ${key}`}>
            <span>{data[key].length}</span>
            <div>
              <h2>{t(title)}</h2>
              <p>{t(detail)}</p>
            </div>
          </article>
        ))}
      </section>
      <div className="dashboard-grid">
        <section className="ready-board">
          <header>
            <p className="section-label">{t("dashboard.executionQueue")}</p>
            <h2>{t("dashboard.readyToStart")}</h2>
          </header>
          {data.readyTasks.length === 0 ? (
            <div className="empty">
              <span>◇</span>
              <h3>{t("dashboard.noTaskTitle")}</h3>
              <p>{t("dashboard.noTaskDetail")}</p>
            </div>
          ) : (
            <ol>
              {data.readyTasks.map((task, index) => {
                const feature = data.features.find(
                  (f) => f.id === task.featureId,
                );
                return (
                  <li key={task.id}>
                    <span className="queue-index">
                      {String(index + 1).padStart(2, "0")}
                    </span>
                    <div>
                      <Link
                        to="/features/$featureId"
                        params={{ featureId: task.featureId }}
                      >
                        {task.title}
                      </Link>
                      <p>
                        {feature?.title} ·{" "}
                        {task.assignee || t("common.unassigned")}
                      </p>
                    </div>
                    <i>{t("common.ready")}</i>
                  </li>
                );
              })}
            </ol>
          )}
        </section>
        <section className="feature-board">
          <header>
            <p className="section-label">{t("dashboard.featureTelemetry")}</p>
            <h2>{t("dashboard.deliveryLines")}</h2>
          </header>
          {data.features.length === 0 ? (
            <div className="empty compact">
              <h3>{t("dashboard.noFeaturesTitle")}</h3>
              <p>{t("dashboard.noFeaturesDetail")}</p>
            </div>
          ) : (
            <div className="feature-table">
              {data.features.map((feature) => (
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
      </div>
    </div>
  );
}

function StateMessage({
  title,
  detail,
  action,
}: {
  title: string;
  detail: string;
  action?: () => void;
}) {
  return (
    <div className="state-message">
      <div className="spinner" />
      <h1>{title}</h1>
      <p>{detail}</p>
      {action && <StateAction action={action} />}
    </div>
  );
}

function StateAction({ action }: { action: () => void }) {
  const { t } = useTranslation();
  return <button onClick={action}>{t("common.retry")}</button>;
}
