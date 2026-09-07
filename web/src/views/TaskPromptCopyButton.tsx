import { Check, ClipboardCopy } from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { getTaskPrompt } from "../api";
import { IconButton } from "./IconButton";

type CopyStatus =
  { case: "idle" } | { case: "copied" } | { case: "failed"; message: string };

// TaskPromptCopyButton hands the task to another agent. It is icon-only with a
// floating status, its accessible name follows the snapshot's plan flag, and the
// copied text always comes from the server, which picks the template.
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
  // Which prompt a task gets is not visible on the task itself, so the outcome
  // names it rather than reporting that something was copied.
  const copiedLabel = hasImplementationPlan
    ? t("inspector.implementationPromptCopied")
    : t("inspector.designPromptCopied");

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
      // shown verbatim. A clipboard failure carries no such detail, so it is
      // reported through the translated text instead of a raw TypeError.
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
