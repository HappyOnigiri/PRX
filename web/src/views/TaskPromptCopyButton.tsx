import { Check, ClipboardCopy } from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { getTaskPrompt } from "../api";
import { IconButton } from "./IconButton";

type CopyStatus =
  { case: "idle" } | { case: "copied" } | { case: "failed"; message: string };

// TaskPromptCopyButton hands the task to another agent. It sits next to the
// task wherever the task is listed, so it is icon-only and reports its outcome
// in a floating status that leaves the surrounding layout alone. The accessible
// name follows the snapshot's plan flag so the reader knows what they are about
// to copy, while the copied text always comes from the server, which decides
// the template from the task as it is at that moment.
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

  function settle(next: CopyStatus) {
    setStatus(next);
    // Like the other copy controls, the outcome clears itself so the button
    // does not keep reporting a result from an earlier click.
    window.setTimeout(() => {
      setStatus({ case: "idle" });
    }, 1600);
  }

  async function copyPrompt() {
    setPending(true);
    setStatus({ case: "idle" });
    try {
      // The server's message names the task or the template at fault, so it is
      // worth showing verbatim. A clipboard failure has no such detail, and
      // navigator.clipboard is simply absent outside a secure context, so those
      // are reported through the translated text instead of a raw TypeError.
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
        {status.case === "copied" && t("inspector.promptCopied")}
        {status.case === "failed" && status.message}
      </span>
    </span>
  );
}
