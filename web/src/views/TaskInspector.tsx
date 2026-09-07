import { Trash2, X } from "lucide-react";
import { useTranslation } from "react-i18next";
import { mutations } from "../api";
import { type PullRequest, type Task } from "../gen/prx/v1/prx_pb";
import { useDomainMutation } from "../hooks";
import { blockedReasonLabel } from "../i18n/domain";
import { CopyableIdentifier } from "./CopyableIdentifier";
import { EntityIcon } from "./EntityIcon";
import { IconButton } from "./IconButton";
import { MutationError } from "./MutationError";
import { PullRequestSection } from "./TaskInspectorPullRequest";
import { ReferencesSection } from "./TaskInspectorReferences";
import { TaskInspectorTaskForm } from "./TaskInspectorTaskForm";
import { type TaskNodeDocument } from "./TaskNode";
import { TaskBlockLabels, TaskStatusBadge } from "./TaskStateBadges";

export interface TaskInspectorProps {
  task: Task;
  tasks: Task[];
  pullRequest: PullRequest | undefined;
  documents: TaskNodeDocument[];
  onPreview: (document: TaskNodeDocument) => void;
  onClose: () => void;
  readOnly?: boolean;
}

function TaskInspectorHeader({
  task,
  onClose,
}: Pick<TaskInspectorProps, "task" | "onClose">) {
  const { t } = useTranslation();
  return (
    <header>
      <div className="inspector-heading">
        {/* どのカードや行とも同じく、状態が見出しの先頭に来る。 */}
        <div className="inspector-heading-line">
          <TaskStatusBadge state={task.displayState} />
          <h2>
            <EntityIcon kind="task" size={16} />
            {task.title}
          </h2>
        </div>
        <CopyableIdentifier
          label={t("common.taskId")}
          value={task.id}
          valueOnly
        />
      </div>
      <IconButton
        icon={X}
        label={t("inspector.close")}
        variant="secondary"
        iconOnly
        onClick={onClose}
      />
    </header>
  );
}

export function TaskInspector({
  task,
  tasks,
  pullRequest,
  documents,
  onPreview,
  onClose,
  readOnly = false,
}: TaskInspectorProps) {
  const { t } = useTranslation();
  const deleteTask = useDomainMutation(mutations.deleteTask);

  return (
    <aside className="inspector" aria-label={t("inspector.label")}>
      <TaskInspectorHeader task={task} onClose={onClose} />
      {/* ステータスは見出しにあるので、このストリップには進行を妨げている事情
          だけが並ぶ。待ち相手はラベルの補足として添える。 */}
      {task.blockLabels.length > 0 && (
        <p className="inspector-blocks">
          <TaskBlockLabels
            labels={task.blockLabels}
            dependencyDetail={blockedReasonLabel(
              task.blockedReason,
              (id) => tasks.find((item) => item.id === id)?.title,
              t,
            )}
          />
        </p>
      )}
      {readOnly && (
        <p className="inspector-read-only">{t("inspector.readOnly")}</p>
      )}
      <TaskInspectorTaskForm task={task} readOnly={readOnly} />
      <PullRequestSection
        taskId={task.id}
        pullRequest={pullRequest}
        readOnly={readOnly}
      />
      <ReferencesSection
        documents={documents}
        onPreview={onPreview}
        readOnly={readOnly}
      />
      {!readOnly && (
        <>
          <IconButton
            icon={Trash2}
            label={t("inspector.deleteTask")}
            variant="danger"
            className="danger-zone"
            onClick={() => {
              if (
                !window.confirm(
                  t("inspector.deleteTaskConfirm", { title: task.title }),
                )
              )
                return;
              void deleteTask
                .mutateAsync(task.id)
                .then(onClose, () => undefined);
            }}
          />
          <MutationError error={deleteTask.error} />
        </>
      )}
    </aside>
  );
}
