import { useTranslation } from "react-i18next";
import { TaskStatus, type Task } from "../gen/prx/v1/prx_pb";
import { taskStatusLabel } from "../i18n/domain";
import { MutationError } from "./MutationError";
import { type TaskDraftController } from "./taskDraft";

export function TaskInspectorTaskForm({
  task,
  controller,
  readOnly = false,
}: {
  task: Task;
  controller: TaskDraftController;
  readOnly?: boolean;
}) {
  const { t } = useTranslation();

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
          <dt>{t("common.assignee")}</dt>
          <dd>{task.assignee || t("common.unassigned")}</dd>
        </div>
      </dl>
    );

  const { draft, setDraft } = controller;
  return (
    <div className="inspector-task-fields">
      <label>
        {t("common.title")}
        <input
          name="title"
          value={draft.title}
          onChange={(event) => {
            setDraft({ ...draft, title: event.target.value });
          }}
        />
      </label>
      <label>
        {t("common.scope")}
        <textarea
          name="scope"
          value={draft.scope}
          onChange={(event) => {
            setDraft({ ...draft, scope: event.target.value });
          }}
        />
      </label>
      <div className="form-row">
        <label>
          {t("common.status")}
          <select
            name="status"
            value={draft.status}
            onChange={(event) => {
              setDraft({ ...draft, status: Number(event.target.value) });
            }}
          >
            {[
              TaskStatus.NOT_STARTED,
              TaskStatus.DESIGNING,
              TaskStatus.IN_PROGRESS,
              TaskStatus.COMPLETED,
              TaskStatus.CLOSED,
            ].map((status) => (
              <option value={status} key={status}>
                {taskStatusLabel(status, t)}
              </option>
            ))}
          </select>
        </label>
        <label>
          {t("common.assignee")}
          <input
            name="assignee"
            value={draft.assignee}
            onChange={(event) => {
              setDraft({ ...draft, assignee: event.target.value });
            }}
          />
        </label>
      </div>
      <MutationError error={controller.error} />
    </div>
  );
}
