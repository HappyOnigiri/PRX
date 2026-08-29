import { useTranslation } from "react-i18next";
import { mutations } from "../api";
import { TaskKind, TaskStatus, type Task } from "../gen/prx/v1/prx_pb";
import { formValue } from "../form";
import { useDomainMutation } from "../hooks";
import { taskStatusLabel } from "../i18n/domain";
import { MutationError } from "./MutationError";

interface TaskInspectorFormProps {
  task: Task;
}

export function TaskInspectorForm({ task }: TaskInspectorFormProps) {
  const { t } = useTranslation();
  const updateTask = useDomainMutation(mutations.updateTask);

  return (
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
      <button disabled={updateTask.isPending}>{t("inspector.saveTask")}</button>
      <MutationError error={updateTask.error} />
    </form>
  );
}
