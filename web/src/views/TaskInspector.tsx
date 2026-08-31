import { Trash2, X } from "lucide-react";
import { useTranslation } from "react-i18next";
import { mutations } from "../api";
import { type PullRequest, type Task } from "../gen/prx/v1/prx_pb";
import { useDomainMutation } from "../hooks";
import {
  blockedReasonLabel,
  taskDisplayStateLabel,
  taskDisplayStateToken,
} from "../i18n/domain";
import { CopyableIdentifier } from "./CopyableIdentifier";
import { IconButton } from "./IconButton";
import { MutationError } from "./MutationError";
import { PullRequestSection } from "./TaskInspectorPullRequest";
import { ReferencesSection } from "./TaskInspectorReferences";
import { TaskInspectorTaskForm } from "./TaskInspectorTaskForm";
import { type TaskNodeDocument } from "./TaskNode";

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
        <h2>{task.title}</h2>
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
      <div
        className={`inspector-state state-${taskDisplayStateToken(task.displayState)}`}
      >
        <i />
        {taskDisplayStateLabel(task.displayState, t)}
        {task.blockedReason && (
          <small>
            {blockedReasonLabel(
              task.blockedReason,
              (id) => tasks.find((item) => item.id === id)?.title,
              t,
            )}
          </small>
        )}
      </div>
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
