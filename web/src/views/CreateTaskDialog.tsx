import { FormEvent } from "react";
import { useTranslation } from "react-i18next";
import { mutations } from "../api";
import { TaskKind } from "../gen/prx/v1/prx_pb";
import { useDomainMutation } from "../hooks";
import { formValue } from "../form";
import { formatError, taskKindLabel } from "../i18n/domain";
import { MutationError } from "./MutationError";

type CreateTaskDialogProps = {
  featureId: string;
  onClose: () => void;
};

export function CreateTaskDialog({
  featureId,
  onClose,
}: CreateTaskDialogProps) {
  const { t } = useTranslation();
  const createTask = useDomainMutation(mutations.createTask);

  async function submitTask(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = new FormData(event.currentTarget);
    try {
      await createTask.mutateAsync({
        featureId,
        title: formValue(form, "title"),
        scope: formValue(form, "scope"),
        kind: Number(form.get("kind")),
        assignee: formValue(form, "assignee"),
      });
    } catch {
      return;
    }
    onClose();
  }

  return (
    <div className="scrim">
      <form
        className="dialog"
        onSubmit={submitTask}
        aria-label={t("taskCreate.formLabel")}
      >
        <header>
          <p>{t("taskCreate.eyebrow")}</p>
          <h2>{t("taskCreate.title")}</h2>
        </header>
        <label>
          {t("common.title")}
          <input
            name="title"
            required
            placeholder={t("taskCreate.titlePlaceholder")}
          />
        </label>
        <label>
          {t("common.scope")}
          <textarea
            name="scope"
            placeholder={t("taskCreate.scopePlaceholder")}
          />
        </label>
        <div className="form-row">
          <label>
            {t("taskCreate.kind")}
            <select name="kind">
              <option value={TaskKind.PULL_REQUEST}>
                {taskKindLabel(TaskKind.PULL_REQUEST, t)}
              </option>
              <option value={TaskKind.MANUAL}>
                {taskKindLabel(TaskKind.MANUAL, t)}
              </option>
            </select>
          </label>
          <label>
            {t("common.assignee")}
            <input
              name="assignee"
              placeholder={t("taskCreate.assigneePlaceholder")}
            />
          </label>
        </div>
        {createTask.error && (
          <p className="form-error">{formatError(createTask.error, t)}</p>
        )}
        <footer>
          <button type="button" className="secondary" onClick={onClose}>
            {t("common.cancel")}
          </button>
          <button disabled={createTask.isPending}>
            {t("taskCreate.submit")}
          </button>
        </footer>
        <MutationError error={createTask.error} />
      </form>
    </div>
  );
}
