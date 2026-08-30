import { useTranslation } from "react-i18next";
import { mutations } from "../api";
import {
  type Dependency,
  type PullRequest,
  type Task,
} from "../gen/prx/v1/prx_pb";
import { useDomainMutation } from "../hooks";
import {
  blockedReasonLabel,
  taskDisplayStateLabel,
  taskDisplayStateToken,
} from "../i18n/domain";
import { CopyableIdentifier } from "./CopyableIdentifier";
import { MutationError } from "./MutationError";
import { DependencySection } from "./TaskInspectorDependencies";
import { ImplementationPlanSection } from "./TaskInspectorImplementationPlan";
import { PullRequestSection } from "./TaskInspectorPullRequest";
import { ReferencesSection } from "./TaskInspectorReferences";
import { TaskInspectorTaskForm } from "./TaskInspectorTaskForm";
import { type TaskNodeDocument } from "./TaskNode";

export interface TaskInspectorProps {
  task: Task;
  tasks: Task[];
  dependencies: Dependency[];
  pullRequest: PullRequest | undefined;
  documents: TaskNodeDocument[];
  onPreview: (document: TaskNodeDocument) => void;
  onClose: () => void;
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
        <CopyableIdentifier label={t("common.taskId")} value={task.id} />
      </div>
      <button
        className="icon-button"
        aria-label={t("inspector.close")}
        onClick={onClose}
      >
        ×
      </button>
    </header>
  );
}

export function TaskInspector({
  task,
  tasks,
  dependencies,
  pullRequest,
  documents,
  onPreview,
  onClose,
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
      <TaskInspectorTaskForm task={task} />
      <PullRequestSection taskId={task.id} pullRequest={pullRequest} />
      <ImplementationPlanSection
        taskId={task.id}
        hasPlan={task.hasImplementationPlan}
      />
      <DependencySection
        taskId={task.id}
        tasks={tasks}
        dependencies={dependencies}
      />
      <ReferencesSection
        taskId={task.id}
        documents={documents}
        onPreview={onPreview}
      />
      <button
        className="danger-zone"
        onClick={() => {
          if (
            !window.confirm(
              t("inspector.deleteTaskConfirm", { title: task.title }),
            )
          )
            return;
          void deleteTask.mutateAsync(task.id).then(onClose, () => undefined);
        }}
      >
        {t("inspector.deleteTask")}
      </button>
      <MutationError error={deleteTask.error} />
    </aside>
  );
}
