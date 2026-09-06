import { ClipboardCopy, Square, SquareCheckBig, X } from "lucide-react";
import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { getBatchPrompt } from "../api";
import { TaskDisplayState, type Task } from "../gen/prx/v1/prx_pb";
import { IconButton } from "./IconButton";

// A task is offered when the server derived both that it has a plan and that
// nothing blocks it. Readiness is a server derivation, so the dialog reads the
// two flags instead of recomputing them from the dependencies it happens to
// hold.
function implementableTasks(tasks: Task[]): Task[] {
  return tasks.filter(
    (task) => task.displayState === TaskDisplayState.DESIGNED && task.ready,
  );
}

type CopyStatus =
  | { case: "idle" }
  | { case: "copied"; count: number }
  | { case: "failed"; message: string };

// BatchPromptDialog hands several tasks to one agent in a single prompt. Like
// the per-task button, the text comes from the server at the moment of the copy,
// so neither an edited template nor a task that changed since the snapshot can
// produce the wrong prompt.
export function BatchPromptDialog({
  featureId,
  tasks,
  onClose,
}: {
  featureId: string;
  tasks: Task[];
  onClose: () => void;
}) {
  const { t } = useTranslation();
  const candidates = useMemo(() => implementableTasks(tasks), [tasks]);
  const [selected, setSelected] = useState<ReadonlySet<string>>(new Set());
  const [status, setStatus] = useState<CopyStatus>({ case: "idle" });
  const [pending, setPending] = useState(false);
  const allSelected =
    candidates.length > 0 && selected.size === candidates.length;

  function toggle(taskId: string) {
    setSelected((current) => {
      const next = new Set(current);
      if (!next.delete(taskId)) next.add(taskId);
      return next;
    });
  }

  function toggleAll() {
    setSelected(
      allSelected ? new Set() : new Set(candidates.map((task) => task.id)),
    );
  }

  async function copyPrompts() {
    // The list order is the order the reader sees, so the copy follows it
    // rather than the order the checkboxes were clicked in.
    const targets = candidates.filter((task) => selected.has(task.id));
    setPending(true);
    setStatus({ case: "idle" });
    try {
      const response = await getBatchPrompt(
        featureId,
        targets.map((task) => task.id),
      );
      try {
        await navigator.clipboard.writeText(response.prompt);
      } catch {
        setStatus({ case: "failed", message: t("batchPrompt.failed") });
        return;
      }
      setStatus({ case: "copied", count: targets.length });
    } catch (error) {
      // The server names the task or the template at fault, so its message is
      // worth showing verbatim.
      setStatus({
        case: "failed",
        message:
          error instanceof Error ? error.message : t("batchPrompt.failed"),
      });
    } finally {
      setPending(false);
    }
  }

  return (
    <div className="scrim" role="presentation">
      <section
        className="dialog batch-prompt-dialog"
        role="dialog"
        aria-modal="true"
        aria-labelledby="batch-prompt-title"
      >
        <header className="batch-prompt-head">
          <div>
            <h2 id="batch-prompt-title">{t("batchPrompt.title")}</h2>
            <p className="dialog-lead">{t("batchPrompt.description")}</p>
          </div>
          <IconButton
            icon={X}
            label={t("common.close")}
            variant="secondary"
            iconOnly
            onClick={onClose}
          />
        </header>
        <BatchPromptTaskList
          candidates={candidates}
          selected={selected}
          allSelected={allSelected}
          onToggle={toggle}
          onToggleAll={toggleAll}
        />
        <footer>
          <p className="batch-prompt-status" aria-live="polite">
            {status.case === "copied" &&
              t("batchPrompt.copied", { selected: status.count })}
            {status.case === "failed" && status.message}
          </p>
          <IconButton
            icon={ClipboardCopy}
            label={t("batchPrompt.copy")}
            variant="primary"
            disabled={pending || selected.size === 0}
            onClick={() => void copyPrompts()}
          />
        </footer>
      </section>
    </div>
  );
}

function BatchPromptTaskList({
  candidates,
  selected,
  allSelected,
  onToggle,
  onToggleAll,
}: {
  candidates: Task[];
  selected: ReadonlySet<string>;
  allSelected: boolean;
  onToggle: (taskId: string) => void;
  onToggleAll: () => void;
}) {
  const { t } = useTranslation();
  if (candidates.length === 0)
    return <p className="batch-prompt-empty">{t("batchPrompt.empty")}</p>;
  return (
    <div className="batch-prompt-body">
      <div className="batch-prompt-toolbar">
        <IconButton
          icon={allSelected ? Square : SquareCheckBig}
          label={
            allSelected ? t("batchPrompt.clearAll") : t("batchPrompt.selectAll")
          }
          variant="quiet"
          size="compact"
          onClick={onToggleAll}
        />
        <span className="batch-prompt-count">
          {t("batchPrompt.selectedCount", {
            selected: selected.size,
            total: candidates.length,
          })}
        </span>
      </div>
      <ul className="batch-prompt-list">
        {candidates.map((task) => (
          <li key={task.id}>
            <label>
              <input
                type="checkbox"
                checked={selected.has(task.id)}
                onChange={() => {
                  onToggle(task.id);
                }}
              />
              <span className="batch-prompt-task-title">{task.title}</span>
              <span className="batch-prompt-task-id">{task.id}</span>
            </label>
          </li>
        ))}
      </ul>
    </div>
  );
}
