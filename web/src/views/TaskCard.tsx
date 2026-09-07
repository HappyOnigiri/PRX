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
import { CopyableIdentifier } from "./CopyableIdentifier";
import { EntityIcon, type EntityKind } from "./EntityIcon";
import { StatusBadge } from "./StatusBadge";
import { TaskPromptCopyButton } from "./TaskPromptCopyButton";

export interface TaskCardProps {
  task: Task;
  // キューは所有者を示したスナップショットより長く残りうるので、どちらも
  // optional にし、タスクを落とさずフォールバックする。
  feature: Feature | undefined;
  project: Project | undefined;
  pullRequest: PullRequest | undefined;
}

// タスクの一覧が現れる場所ではこのカード 1 つがタスクを示す。おかげで概要の
// キューと検索結果は、読むときも眺めるときも同じものになる。
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
          {/* 状態を行頭に置くことで、各行末のバッジを探し回らずに状態の列を
              1 本追うだけで済む。 */}
          <StatusBadge
            className={`state-${taskDisplayStateToken(task.displayState)}`}
            label={taskDisplayStateLabel(task.displayState, t)}
          />
          <EntityIcon kind="task" size={15} />
          <Link
            to="/features/$featureId"
            params={{ featureId: task.featureId }}
          >
            {task.title}
          </Link>
          {/* ID はエージェントや検索に渡すものなので、グラフ上と同じく名前の
              隣に置き、クリックでコピーできるようにする。 */}
          <CopyableIdentifier
            label={t("common.taskId")}
            value={task.id}
            valueOnly
          />
        </p>
        {/* 同じ大きさの値が並ぶと 1 つの文に見えるので、どのフィールドの値かを
            アイコンで示す。グリフを読めない支援技術のために名前は残す。 */}
        <dl className="task-card-meta">
          <div>
            <dt>{t("taskCard.project")}</dt>
            <ProjectValue project={project} />
          </div>
          <div>
            <dt>{t("taskCard.feature")}</dt>
            <FeatureValue feature={feature} />
          </div>
          {/* 未割り当てのタスクは空の担当者を示しても意味がないので、プレース
              ホルダを出さずペアごと落とす。 */}
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
      <TaskPromptCopyButton
        taskId={task.id}
        hasImplementationPlan={task.hasImplementationPlan}
        size="compact"
      />
    </li>
  );
}

// feature は必ずプロジェクトに属するので、見つからないのはスナップショットが
// 運んでいないということ。リンク先がないので値はただのテキストにする。
function ProjectValue({ project }: { project: Project | undefined }) {
  const { t } = useTranslation();
  if (!project)
    return (
      <MetaValue kind="project">
        <span className="task-card-meta-name">{t("taskCard.unknown")}</span>
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

// feature がなければ示す名前も行き先もないので、組み立てられないページへの
// リンクにせず、値はただのテキストにする。
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

// pull request は PRX ではなく GitHub 上にあるので、値はサーバーが記録した正規
// の URL を開き、その隣にレビュー状態を示す。
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
