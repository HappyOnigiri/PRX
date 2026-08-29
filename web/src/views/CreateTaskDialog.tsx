import type { SyntheticEvent } from "react";
import { useTranslation } from "react-i18next";
import { mutations } from "../api";
import { useDomainMutation } from "../hooks";
import { formValue } from "../form";
import { formatError } from "../i18n/domain";
import { CreateTaskFields } from "./CreateTaskFields";
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
        <CreateTaskFields />
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
