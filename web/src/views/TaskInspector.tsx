import { useTranslation } from "react-i18next";
import { mutations } from "../api";
import {
  TaskKind,
  TaskStatus,
  type Dependency,
  type PullRequest,
  type Task,
} from "../gen/prx/v1/prx_pb";
import { useDomainMutation } from "../hooks";
import { formValue } from "../form";
import {
  blockedReasonLabel,
  taskDisplayStateLabel,
  taskDisplayStateToken,
  taskStatusLabel,
} from "../i18n/domain";
import { type TaskNodeDocument } from "./TaskNode";
import { MutationError } from "./MutationError";
import { DependencySection } from "./TaskInspectorDependencies";
import { PullRequestSection } from "./TaskInspectorPullRequest";
import { ReferencesSection } from "./TaskInspectorReferences";

export interface TaskInspectorProps {
  task: Task;
  tasks: Task[];
  dependencies: Dependency[];
  pullRequest: PullRequest | undefined;
  documents: TaskNodeDocument[];
  onPreview: (document: TaskNodeDocument) => void;
  onClose: () => void;
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
  const updateTask = useDomainMutation(mutations.updateTask);
  const deleteTask = useDomainMutation(mutations.deleteTask);

  return (
    <aside className="inspector" aria-label={t("inspector.label")}>
      <header>
        <h2>{task.title}</h2>
        <button
          className="icon-button"
          aria-label={t("inspector.close")}
          onClick={onClose}
        >
          ×
        </button>
      </header>
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
      <form
        onSubmit={(event) => {
          event.preventDefault();
          const form = new FormData(event.currentTarget);
          updateTask.mutate({
            id: task.id,
            title: formValue(form, "title"),
            scope: formValue(form, "scope"),
            status: Number(form.get("status")),
            assignee: formValue(form, "assignee"),
          });
        }}
      >
        <label>
          {t("common.title")}
          <input name="title" defaultValue={task.title} />
        </label>
        <label>
          {t("common.scope")}
          <textarea name="scope" defaultValue={task.scope} />
        </label>
        <div className="form-row">
          <label>
            {t("common.status")}
            <select name="status" defaultValue={task.status}>
              <option value={TaskStatus.PLANNED}>
                {taskStatusLabel(TaskStatus.PLANNED, t)}
              </option>
              <option value={TaskStatus.IN_PROGRESS}>
                {taskStatusLabel(TaskStatus.IN_PROGRESS, t)}
              </option>
              {/* A PR task completes when its pull request merges. */}
              {task.kind !== TaskKind.PULL_REQUEST && (
                <option value={TaskStatus.COMPLETED}>
                  {taskStatusLabel(TaskStatus.COMPLETED, t)}
                </option>
              )}
              <option value={TaskStatus.CANCELLED}>
                {taskStatusLabel(TaskStatus.CANCELLED, t)}
              </option>
            </select>
          </label>
          <label>
            {t("common.assignee")}
            <input name="assignee" defaultValue={task.assignee} />
          </label>
        </div>
        <button disabled={updateTask.isPending}>
          {t("inspector.saveTask")}
        </button>
        <MutationError error={updateTask.error} />
      </form>
      <PullRequestSection taskId={task.id} pullRequest={pullRequest} />
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
