import { Link } from "@tanstack/react-router";
import { RefreshCw, RotateCcw } from "lucide-react";
import { useTranslation } from "react-i18next";
import { mutations } from "../api";
import { isActiveFeature } from "../feature-status";
import type { Task } from "../gen/prx/v1/prx_pb";
import { useDomainMutation, useSnapshot } from "../hooks";
import { formatError } from "../i18n/domain";
import { useAutoSyncStatus } from "../sync-status";
import { filterTaskSearchResults } from "../task-search";
import { IconButton } from "./IconButton";
import { TaskCard } from "./TaskCard";

// 件数はタスクのステータスとブロックラベルから数えるので、リンク先のクエリも
// 同じ軸で書く。pull request 自身の状態を指す github-status では件数と一覧が
// 食い違う。
const queueNames = [
  [
    "readyTasks",
    "queue-ready",
    "dashboard.queues.ready.title",
    "task-status:ready",
  ],
  [
    "reviewWaitingTasks",
    "queue-review-waiting",
    "dashboard.queues.review.title",
    "task-status:in-review",
  ],
  [
    "conflictTasks",
    "queue-conflict",
    "dashboard.queues.conflicts.title",
    "block:conflict",
  ],
  [
    "syncErrorTasks",
    "queue-sync-error",
    "dashboard.queues.syncError.title",
    "github-status:error",
  ],
] as const;
const alwaysVisibleQueueKeys = new Set(["readyTasks", "reviewWaitingTasks"]);
type ProjectedQueues = Record<(typeof queueNames)[number][0], Task[]>;

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
  if (error)
    return (
      <StateMessage
        title={t("dashboard.errorTitle")}
        detail={formatError(error, t)}
        action={() => void refetch()}
      />
    );
  const features = data.features.filter(isActiveFeature);
  const featureIds = new Set(features.map((feature) => feature.id));
  const syncErrorTaskIds = new Set(
    filterTaskSearchResults(data, {
      qualifiers: [{ type: "github-status", value: "error" }],
      terms: [],
    }).map(({ task }) => task.id),
  );
  const pullRequests = new Map(data.pullRequests.map((pr) => [pr.taskId, pr]));
  const projected = {
    readyTasks: data.readyTasks.filter((task) =>
      featureIds.has(task.featureId),
    ),
    reviewWaitingTasks: data.reviewWaitingTasks.filter((task) =>
      featureIds.has(task.featureId),
    ),
    conflictTasks: data.conflictTasks.filter((task) =>
      featureIds.has(task.featureId),
    ),
    syncErrorTasks: data.tasks.filter(
      (task) => featureIds.has(task.featureId) && syncErrorTaskIds.has(task.id),
    ),
  };
  return (
    <div className="dashboard">
      <header className="page-head">
        <div>
          <h1>
            {t("dashboard.titleStart")}
            <em>{t("dashboard.titleEmphasis")}</em>
          </h1>
        </div>
        <div className="page-head-status">
          <SyncStatus />
        </div>
      </header>
      <QueueStrip projected={projected} />
      <div className="dashboard-grid">
        <section
          className="ready-board"
          aria-label={t("dashboard.readyToStart")}
        >
          {projected.readyTasks.length === 0 ? (
            <div className="empty">
              <span>◇</span>
              <h3>{t("dashboard.noTaskTitle")}</h3>
              <p>{t("dashboard.noTaskDetail")}</p>
            </div>
          ) : (
            <ol className="task-card-list">
              {projected.readyTasks.map((task) => {
                const feature = features.find(
                  (item) => item.id === task.featureId,
                );
                return (
                  <TaskCard
                    key={task.id}
                    task={task}
                    feature={feature}
                    project={data.projects.find(
                      (item) => item.id === feature?.projectId,
                    )}
                    pullRequest={pullRequests.get(task.id)}
                  />
                );
              })}
            </ol>
          )}
        </section>
      </div>
    </div>
  );
}

function QueueStrip({ projected }: { projected: ProjectedQueues }) {
  const { t } = useTranslation();
  return (
    <section className="queue-strip" aria-label={t("dashboard.roadmapStatus")}>
      {queueNames
        .filter(
          ([key]) =>
            alwaysVisibleQueueKeys.has(key) || projected[key].length > 0,
        )
        .map(([key, className, title, query]) => (
          <Link
            key={key}
            to="/tasks"
            search={{ q: query }}
            className={`queue-meter ${className}`}
          >
            <span>{projected[key].length}</span>
            <h2>{t(title)}</h2>
          </Link>
        ))}
    </section>
  );
}

function SyncStatus() {
  const { t } = useTranslation();
  const status = useAutoSyncStatus();
  const sync = useDomainMutation(() => mutations.sync());
  const pending = sync.isPending || status.checking;
  return (
    <div className="dashboard-sync">
      <IconButton
        icon={RefreshCw}
        label={pending ? t("dashboard.syncingNow") : t("dashboard.syncNow")}
        variant="secondary"
        busy={pending}
        disabled={pending}
        onClick={() => {
          sync.mutate(undefined);
        }}
      />
      {sync.error && (
        <small className="dashboard-sync-error" role="alert">
          {formatError(sync.error, t)}
        </small>
      )}
    </div>
  );
}

export function StateMessage({
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
  return (
    <IconButton
      icon={RotateCcw}
      label={t("common.retry")}
      variant="secondary"
      onClick={action}
    />
  );
}
