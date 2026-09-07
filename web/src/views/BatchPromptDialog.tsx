import { ClipboardCopy, Square, SquareCheckBig, X } from "lucide-react";
import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { getBatchPrompt } from "../api";
import { type Task } from "../gen/prx/v1/prx_pb";
import {
  batchCandidates,
  isSelectable,
  prunedSelection,
  type BatchCandidate,
} from "./batchPromptTasks";
import { IconButton } from "./IconButton";

type CopyStatus =
  | { case: "idle" }
  | { case: "copied"; count: number }
  | { case: "failed"; message: string };

// BatchPromptDialog hands several tasks to one agent in a single prompt, with
// the text rendered by the server at the moment of the copy.
// See docs/design/agent-prompts.md.
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
  // Whether the blocked tasks are offered is a choice about this one handover,
  // not a preference the reader keeps, so it starts closed on every open.
  const [includeBlocked, setIncludeBlocked] = useState(false);
  const candidates = useMemo(
    () => batchCandidates(tasks, includeBlocked),
    [tasks, includeBlocked],
  );
  const [selected, setSelected] = useState<ReadonlySet<string>>(new Set());
  const [status, setStatus] = useState<CopyStatus>({ case: "idle" });
  const [pending, setPending] = useState(false);
  const allSelected =
    candidates.length > 0 && selected.size === candidates.length;

  function toggle(taskId: string) {
    setSelected((current) => {
      const next = new Set(current);
      // Dropping a task strands whatever was stacked on it, so the selection is
      // pruned rather than left describing work with no base.
      if (next.delete(taskId)) return prunedSelection(candidates, next);
      next.add(taskId);
      return next;
    });
  }

  function toggleAll() {
    // Every offered task can be selected at once: the offer already excludes
    // anything whose blockers the batch cannot carry.
    setSelected(
      allSelected
        ? new Set()
        : new Set(candidates.map((candidate) => candidate.task.id)),
    );
  }

  function changeIncludeBlocked(include: boolean) {
    setIncludeBlocked(include);
    setSelected((current) =>
      prunedSelection(batchCandidates(tasks, include), current),
    );
  }

  async function copyPrompts() {
    // The list order is the order the reader sees, so the copy follows it
    // rather than the order the checkboxes were clicked in.
    const targets = candidates.filter((candidate) =>
      selected.has(candidate.task.id),
    );
    setPending(true);
    setStatus({ case: "idle" });
    try {
      const response = await getBatchPrompt(
        featureId,
        targets.map((candidate) => candidate.task.id),
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
          includeBlocked={includeBlocked}
          onToggle={toggle}
          onToggleAll={toggleAll}
          onIncludeBlockedChange={changeIncludeBlocked}
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
  includeBlocked,
  onToggle,
  onToggleAll,
  onIncludeBlockedChange,
}: {
  candidates: BatchCandidate[];
  selected: ReadonlySet<string>;
  allSelected: boolean;
  includeBlocked: boolean;
  onToggle: (taskId: string) => void;
  onToggleAll: () => void;
  onIncludeBlockedChange: (include: boolean) => void;
}) {
  const { t } = useTranslation();
  // A feature with nothing ready has nothing the dependent tasks could stack
  // on either, so the option to reveal them is left out with the list.
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
        {/* The blocked tasks are a checkbox rather than a row control: it turns
            an option on for the whole list instead of joining the selection the
            rows carry. */}
        <label className="batch-prompt-include-blocked">
          <input
            type="checkbox"
            checked={includeBlocked}
            onChange={(event) => {
              onIncludeBlockedChange(event.target.checked);
            }}
          />
          {t("batchPrompt.includeBlocked")}
        </label>
        <span className="batch-prompt-count">
          {t("batchPrompt.selectedCount", {
            selected: selected.size,
            total: candidates.length,
          })}
        </span>
      </div>
      <ul className="batch-prompt-list">
        {candidates.map((candidate) => (
          <li key={candidate.task.id}>
            {/* The row itself is the control, and aria-pressed carries the
                selection that the accent edge and fill state by appearance.
                See docs/design/webui.md. */}
            <button
              type="button"
              className="batch-prompt-task"
              aria-pressed={selected.has(candidate.task.id)}
              // A task whose blockers are not being handed over has no base to
              // start from, so it stays out of reach until they are selected.
              disabled={!isSelectable(candidate, selected)}
              onClick={() => {
                onToggle(candidate.task.id);
              }}
            >
              <span className="batch-prompt-task-title">
                {candidate.task.title}
              </span>
              {candidate.pendingBlockerIds.length > 0 && (
                <span className="batch-prompt-task-after">
                  {t("batchPrompt.afterTasks", {
                    tasks: candidate.pendingBlockerIds.join(", "),
                  })}
                </span>
              )}
              <span className="batch-prompt-task-id">{candidate.task.id}</span>
            </button>
          </li>
        ))}
      </ul>
    </div>
  );
}
