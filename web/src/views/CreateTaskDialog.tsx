import { Plus, X } from "lucide-react";
import { useRef, useState, type SyntheticEvent } from "react";
import { useTranslation } from "react-i18next";
import { mutations } from "../api";
import { formValue } from "../form";
import { useDomainMutation, useSnapshotRefresh } from "../hooks";
import { formatError } from "../i18n/domain";
import { dependencyPair, type PendingDependency } from "./dependencyGraph";
import { IconButton } from "./IconButton";
import { MutationError } from "./MutationError";

interface CreateTaskInput {
  featureId: string;
  title: string;
  scope: string;
  assignee: string;
}

interface CreateTaskDialogProps {
  featureId: string;
  onClose: () => void;
  dependency?: PendingDependency;
  dependencyTitle?: string;
}

export function CreateTaskDialog({
  featureId,
  onClose,
  dependency,
  dependencyTitle,
}: CreateTaskDialogProps) {
  const { t } = useTranslation();
  const refreshSnapshot = useSnapshotRefresh();
  const [dependencyFailed, setDependencyFailed] = useState(false);
  // 依存の追加だけが失敗したときの再送で、タスクを二重に作らないための控え。
  const createdTaskId = useRef<string>(undefined);
  const createTask = useDomainMutation(async (input: CreateTaskInput) => {
    const created = createdTaskId.current;
    // 再送では作成をやり直さないので、その間に直したフォームの値は更新で送る。
    if (created)
      await mutations.updateTask({
        id: created,
        title: input.title,
        scope: input.scope,
        assignee: input.assignee,
      });
    const taskId = created ?? (await mutations.createTask(input)).task?.id;
    createdTaskId.current = taskId;
    if (dependency && taskId) {
      const { blocker, blocked } = dependencyPair(dependency, taskId);
      try {
        await mutations.addDependency(blocker, blocked);
      } catch (error) {
        // タスクだけが書き込まれた状態なので、依存が欠けたノードをグラフに出す。
        setDependencyFailed(true);
        await refreshSnapshot();
        throw error;
      }
    }
  });

  async function submitTask(event: SyntheticEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = new FormData(event.currentTarget);
    try {
      await createTask.mutateAsync({
        featureId,
        title: formValue(form, "title"),
        scope: formValue(form, "scope"),
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
        {dependency && (
          <p className="dialog-lead">
            {t(
              dependency.direction === "blocks"
                ? "taskCreate.dependencyBlocks"
                : "taskCreate.dependencyBlockedBy",
              { title: dependencyTitle ?? dependency.taskId },
            )}
          </p>
        )}
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
        <label>
          {t("common.assignee")}
          <input
            name="assignee"
            placeholder={t("taskCreate.assigneePlaceholder")}
          />
        </label>
        {dependencyFailed && (
          <p className="form-error">{t("taskCreate.dependencyFailed")}</p>
        )}
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
