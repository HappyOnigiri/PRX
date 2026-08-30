import { Plus, X } from "lucide-react";
import type { SyntheticEvent } from "react";
import { useTranslation } from "react-i18next";
import { mutations } from "../api";
import { formValue } from "../form";
import { TaskKind } from "../gen/prx/v1/prx_pb";
import { useDomainMutation } from "../hooks";
import { formatError, taskKindLabel } from "../i18n/domain";
import { IconButton } from "./IconButton";
import { MutationError } from "./MutationError";

interface CreateTaskDialogProps {
  featureId: string;
  onClose: () => void;
}

export function CreateTaskDialog({
  featureId,
  onClose,
}: CreateTaskDialogProps) {
  const { t } = useTranslation();
  const createTask = useDomainMutation(mutations.createTask);

  async function submitTask(event: SyntheticEvent<HTMLFormElement>) {
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
          <IconButton
            icon={X}
            label={t("common.cancel")}
            variant="secondary"
            onClick={onClose}
          />
          <IconButton
            icon={Plus}
            label={t("taskCreate.submit")}
            variant="primary"
            type="submit"
            disabled={createTask.isPending}
          />
        </footer>
        <MutationError error={createTask.error} />
      </form>
    </div>
  );
}
