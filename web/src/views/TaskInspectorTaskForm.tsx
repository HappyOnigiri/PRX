import { Save } from "lucide-react";
import { useTranslation } from "react-i18next";
import { mutations } from "../api";
import { formValue } from "../form";
import { TaskStatus, type Task } from "../gen/prx/v1/prx_pb";
import { useDomainMutation } from "../hooks";
import { taskStatusLabel } from "../i18n/domain";
import { IconButton } from "./IconButton";
import { MutationError } from "./MutationError";

export function TaskInspectorTaskForm({
  task,
  readOnly = false,
}: {
  task: Task;
  readOnly?: boolean;
}) {
  const { t } = useTranslation();
  const updateTask = useDomainMutation(mutations.updateTask);

  if (readOnly)
    return (
      <dl className="inspector-values">
        <div>
          <dt>{t("common.title")}</dt>
          <dd>{task.title}</dd>
        </div>
        <div>
          <dt>{t("common.scope")}</dt>
          <dd>{task.scope || t("inspector.notSet")}</dd>
        </div>
        <div>
          <dt>{t("common.status")}</dt>
          <dd>{taskStatusLabel(task.status, t)}</dd>
        </div>
        <div>
          <dt>{t("common.assignee")}</dt>
          <dd>{task.assignee || t("common.unassigned")}</dd>
        </div>
      </dl>
    );

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
            <option value={TaskStatus.AUTO}>
              {taskStatusLabel(TaskStatus.AUTO, t)}
            </option>
            <option value={TaskStatus.NOT_STARTED}>
              {taskStatusLabel(TaskStatus.NOT_STARTED, t)}
            </option>
            <option value={TaskStatus.IN_PROGRESS}>
              {taskStatusLabel(TaskStatus.IN_PROGRESS, t)}
            </option>
            <option value={TaskStatus.COMPLETED}>
              {taskStatusLabel(TaskStatus.COMPLETED, t)}
            </option>
            <option value={TaskStatus.CLOSED}>
              {taskStatusLabel(TaskStatus.CLOSED, t)}
            </option>
          </select>
        </label>
        <label>
          {t("common.assignee")}
          <input name="assignee" defaultValue={task.assignee} />
        </label>
      </div>
      <IconButton
        icon={Save}
        label={t("inspector.saveTask")}
        variant="primary"
        type="submit"
        disabled={updateTask.isPending}
      />
      <MutationError error={updateTask.error} />
    </form>
  );
}
