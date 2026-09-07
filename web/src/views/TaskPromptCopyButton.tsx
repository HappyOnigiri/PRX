import { Check, ClipboardCopy } from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { getTaskPrompt } from "../api";
import { IconButton } from "./IconButton";

type CopyStatus =
  { case: "idle" } | { case: "copied" } | { case: "failed"; message: string };

// TaskPromptCopyButton はタスクを別のエージェントに渡す。アイコンのみで状態を
// 浮かせて表示し、アクセシブル名はスナップショットの plan フラグに従う。コピー
// されるテキストは常にサーバー由来で、テンプレートもサーバーが選ぶ。
export function TaskPromptCopyButton({
  taskId,
  hasImplementationPlan,
  size = "standard",
}: {
  taskId: string;
  hasImplementationPlan: boolean;
  size?: "standard" | "compact";
}) {
  const { t } = useTranslation();
  const [status, setStatus] = useState<CopyStatus>({ case: "idle" });
  const [pending, setPending] = useState(false);

  const label = hasImplementationPlan
    ? t("inspector.copyImplementationPrompt")
    : t("inspector.copyDesignPrompt");
  // どのプロンプトになるかはタスク自体からは見えないので、結果は何かをコピー
  // したとだけ言わずプロンプトの名前を示す。
  const copiedLabel = hasImplementationPlan
    ? t("inspector.implementationPromptCopied")
    : t("inspector.designPromptCopied");

  function settle(next: CopyStatus) {
    setStatus(next);
    // 他のコピー操作と同じく結果表示は自動で消え、前回のクリックの結果を
    // ボタンが示し続けないようにする。
    window.setTimeout(() => {
      setStatus({ case: "idle" });
    }, 1600);
  }

  async function copyPrompt() {
    setPending(true);
    setStatus({ case: "idle" });
    try {
      // サーバーのメッセージは原因となったタスクやテンプレートを名指しするので
      // そのまま見せる。クリップボードの失敗にはそうした情報がないので、生の
      // TypeError ではなく翻訳済みのテキストで伝える。
      const response = await getTaskPrompt(taskId);
      try {
        await navigator.clipboard.writeText(response.prompt);
      } catch {
        settle({ case: "failed", message: t("inspector.promptFailed") });
        return;
      }
      settle({ case: "copied" });
    } catch (error) {
      settle({
        case: "failed",
        message:
          error instanceof Error ? error.message : t("inspector.promptFailed"),
      });
    } finally {
      setPending(false);
    }
  }

  return (
    <span className="task-prompt-copy">
      <IconButton
        icon={status.case === "copied" ? Check : ClipboardCopy}
        label={label}
        variant="secondary"
        size={size}
        iconOnly
        className={
          status.case === "copied"
            ? "task-prompt-copy-button is-copied"
            : "task-prompt-copy-button"
        }
        type="button"
        disabled={pending}
        onClick={() => void copyPrompt()}
      />
      <span className="task-prompt-copy-status" aria-live="polite">
        {status.case === "copied" && copiedLabel}
        {status.case === "failed" && status.message}
      </span>
    </span>
  );
}
