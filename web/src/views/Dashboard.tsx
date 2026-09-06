import { Link } from "@tanstack/react-router";
import { RefreshCw, RotateCcw } from "lucide-react";
import { useTranslation } from "react-i18next";
import { mutations } from "../api";
import { isActiveFeature } from "../feature-status";
import type { Feature, Project, Task } from "../gen/prx/v1/prx_pb";
import { useDomainMutation, useSnapshot } from "../hooks";
import {
  formatError,
  taskDisplayStateLabel,
  taskDisplayStateToken,
} from "../i18n/domain";
import { useAutoSyncStatus } from "../sync-status";
import { filterTaskSearchResults } from "../task-search";
import { EntityIcon, type EntityKind } from "./EntityIcon";
import { IconButton } from "./IconButton";
import { TaskPromptCopyButton } from "./TaskPromptCopyButton";

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
    "github-status:review-waiting",
  ],
  [
    "conflictTasks",
    "queue-conflict",
    "dashboard.queues.conflicts.title",
    "github-status:conflict",
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
            <ol>
              {projected.readyTasks.map((task) => (
                <QueueRow
                  key={task.id}
                  task={task}
                  features={features}
                  projects={data.projects}
                />
              ))}
            </ol>
          )}
        </section>
      </div>
    </div>
  );
}

function QueueRow({
  task,
  features,
  projects,
}: {
  task: Task;
  features: Feature[];
  projects: Project[];
}) {
  const { t } = useTranslation();
  const feature = features.find((item) => item.id === task.featureId);
  const project = projects.find((item) => item.id === feature?.projectId);
  return (
    <li>
      <div>
        <p className="queue-title">
          <EntityIcon kind="task" size={15} />
          <Link
            to="/features/$featureId"
            params={{ featureId: task.featureId }}
          >
            {task.title}
          </Link>
        </p>
        {/* Three values of the same size read as one sentence, so an icon
            marks which field each one belongs to. The names stay for
            assistive technology, which cannot read a glyph. */}
        <dl className="queue-meta">
          <div>
            <dt>{t("dashboard.metaProject")}</dt>
            <MetaValue
              kind="project"
              value={project?.title ?? t("project.unassignedTitle")}
            />
          </div>
          <div>
            <dt>{t("dashboard.metaFeature")}</dt>
            <MetaValue kind="feature" value={feature?.title ?? ""} />
          </div>
          {/* An unassigned task says nothing by naming an empty owner, so the
              pair is dropped instead of carrying a placeholder. */}
          {task.assignee && (
            <div>
              <dt>{t("dashboard.metaAssignee")}</dt>
              <MetaValue kind="assignee" value={task.assignee} />
            </div>
          )}
        </dl>
      </div>
      <i className={`state-${taskDisplayStateToken(task.displayState)}`}>
        {taskDisplayStateLabel(task.displayState, t)}
      </i>
      <TaskPromptCopyButton
        taskId={task.id}
        hasImplementationPlan={task.hasImplementationPlan}
        size="compact"
      />
    </li>
  );
}

function MetaValue({ kind, value }: { kind: EntityKind; value: string }) {
  return (
    <dd>
      <EntityIcon kind={kind} size={13} />
      <span className="queue-meta-name">{value}</span>
    </dd>
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
