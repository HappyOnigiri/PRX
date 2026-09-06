import { Link } from "@tanstack/react-router";
import { ExternalLink } from "lucide-react";
import type { ReactNode } from "react";
import { useTranslation } from "react-i18next";
import type { Feature, Project, PullRequest, Task } from "../gen/prx/v1/prx_pb";
import {
  pullRequestDisplayStateLabel,
  taskDisplayStateLabel,
  taskDisplayStateToken,
} from "../i18n/domain";
import { EntityIcon, type EntityKind } from "./EntityIcon";
import { TaskPromptCopyButton } from "./TaskPromptCopyButton";

export interface TaskCardProps {
  task: Task;
  // A queue can outlive the snapshot that named the owners, so both stay
  // optional and the card falls back rather than dropping the task.
  feature: Feature | undefined;
  project: Project | undefined;
  pullRequest: PullRequest | undefined;
}

// One card presents a task wherever a list of tasks appears, so the overview
// queue and the search results stay the same object to read and to scan.
export function TaskCard({
  task,
  feature,
  project,
  pullRequest,
}: TaskCardProps) {
  const { t } = useTranslation();
  return (
    <li className="task-card">
      <div>
        <p className="task-card-title">
          <EntityIcon kind="task" size={15} />
          <Link
            to="/features/$featureId"
            params={{ featureId: task.featureId }}
          >
            {task.title}
          </Link>
        </p>
        {/* Values of the same size read as one sentence, so an icon marks
            which field each one belongs to. The names stay for assistive
            technology, which cannot read a glyph. */}
        <dl className="task-card-meta">
          <div>
            <dt>{t("taskCard.project")}</dt>
            <ProjectValue project={project} />
          </div>
          <div>
            <dt>{t("taskCard.feature")}</dt>
            <FeatureValue feature={feature} />
          </div>
          {/* An unassigned task says nothing by naming an empty owner, so the
              pair is dropped instead of carrying a placeholder. */}
          {task.assignee && (
            <div>
              <dt>{t("taskCard.assignee")}</dt>
              <MetaValue kind="assignee">
                <span className="task-card-meta-name">{task.assignee}</span>
              </MetaValue>
            </div>
          )}
          {pullRequest && (
            <div>
              <dt>{t("taskCard.pullRequest")}</dt>
              <PullRequestValue pullRequest={pullRequest} />
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

// A task whose feature has no project belongs to the unaffiliated list, which
// is a page of its own rather than a project route.
function ProjectValue({ project }: { project: Project | undefined }) {
  const { t } = useTranslation();
  if (!project)
    return (
      <MetaValue kind="project">
        <Link
          to="/projects/unassigned"
          search={{ features: "active" }}
          className="task-card-meta-name"
        >
          {t("project.unassignedTitle")}
        </Link>
      </MetaValue>
    );
  return (
    <MetaValue kind="project">
      <Link
        to="/projects/$projectId"
        params={{ projectId: project.id }}
        search={{ features: "active" }}
        className="task-card-meta-name"
      >
        {project.title}
      </Link>
    </MetaValue>
  );
}

// Without the feature there is nothing to name and nowhere to go, so the value
// stays plain text instead of a link to a page that cannot be built.
function FeatureValue({ feature }: { feature: Feature | undefined }) {
  const { t } = useTranslation();
  if (!feature)
    return (
      <MetaValue kind="feature">
        <span className="task-card-meta-name">{t("taskCard.unknown")}</span>
      </MetaValue>
    );
  return (
    <MetaValue kind="feature">
      <Link
        to="/features/$featureId"
        params={{ featureId: feature.id }}
        className="task-card-meta-name"
      >
        {feature.title}
      </Link>
    </MetaValue>
  );
}

// The pull request lives on GitHub rather than in PRX, so the value opens the
// canonical URL the server recorded and states the review state next to it.
function PullRequestValue({ pullRequest }: { pullRequest: PullRequest }) {
  const { t } = useTranslation();
  const host =
    pullRequest.host && pullRequest.host !== "github.com"
      ? `${pullRequest.host}/`
      : "";
  return (
    <dd className="task-card-pr">
      <a href={pullRequest.url} target="_blank" rel="noreferrer">
        <EntityIcon kind="pullRequest" size={13} />
        <span className="task-card-meta-name">
          {`${host}${pullRequest.owner}/${pullRequest.repository} #${String(pullRequest.number)}`}
        </span>
        <ExternalLink aria-hidden="true" focusable="false" size={13} />
      </a>
      <small>{pullRequestDisplayStateLabel(pullRequest.displayState, t)}</small>
      {pullRequest.syncError && (
        <em title={pullRequest.syncError}>{pullRequest.syncError}</em>
      )}
    </dd>
  );
}

function MetaValue({
  kind,
  children,
}: {
  kind: EntityKind;
  children: ReactNode;
}) {
  return (
    <dd>
      <EntityIcon kind={kind} size={13} />
      {children}
    </dd>
  );
}
