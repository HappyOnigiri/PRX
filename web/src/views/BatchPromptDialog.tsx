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

// BatchPromptDialog は複数のタスクを 1 つのプロンプトにまとめて
// エージェントへ渡す。本文はコピー時にサーバーが生成する。
// docs/design/agent-prompts.md を参照。
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
  // ブロック中タスクを候補に含めるかはこの受け渡し限りの判断で、
  // 保持する設定ではないため、開くたびにオフから始める。
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
      // タスクを外すとその上に積まれた作業が土台を失うので、
      // 選択を残さず刈り込む。
      if (next.delete(taskId)) return prunedSelection(candidates, next);
      next.add(taskId);
      return next;
    });
  }

  function toggleAll() {
    // 候補はすべて同時に選べる。バッチで運べないブロッカーを持つものは
    // 候補の時点で除外済み。
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
    // リストの並びが読み手に見える順序なので、コピーはチェックした順ではなく
    // その並びに従う。
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
      // サーバーは原因となったタスクやテンプレートを示すので、
      // メッセージはそのまま表示する価値がある。
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
  // 着手可能なタスクがない feature では依存側が積む土台もないので、
  // それらを表示する選択肢もリストごと省く。
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
        {/* ブロック中タスクは行の操作ではなくチェックボックスにする。行が持つ
            選択に加わるのではなく、リスト全体の表示を切り替えるため。 */}
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
            {/* 行そのものが操作要素で、見た目ではアクセント枠と塗りが示す選択を
                aria-pressed が伝える。docs/design/webui.md を参照。 */}
            <button
              type="button"
              className="batch-prompt-task"
              aria-pressed={selected.has(candidate.task.id)}
              // ブロッカーを一緒に渡さないタスクは着手の土台がないので、
              // それらが選ばれるまで選択できない。
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
