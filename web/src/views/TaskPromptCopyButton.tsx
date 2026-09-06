import { Check, ClipboardCopy } from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { getTaskPrompt } from "../api";
import { type Task } from "../gen/prx/v1/prx_pb";
import { IconButton } from "./IconButton";

type CopyStatus =
  { case: "idle" } | { case: "copied" } | { case: "failed"; message: string };

// TaskPromptCopyButton hands the task to another agent. The label follows the
// snapshot's plan flag so the reader knows what they are about to copy, while
// the copied text always comes from the server, which decides the template from
// the task as it is at that moment.
export function TaskPromptCopyButton({ task }: { task: Task }) {
  const { t } = useTranslation();
  const [status, setStatus] = useState<CopyStatus>({ case: "idle" });
  const [pending, setPending] = useState(false);

  const label = task.hasImplementationPlan
    ? t("inspector.copyImplementationPrompt")
    : t("inspector.copyDesignPrompt");

  async function copyPrompt() {
    setPending(true);
    setStatus({ case: "idle" });
    try {
      const response = await getTaskPrompt(task.id);
      await navigator.clipboard.writeText(response.prompt);
      setStatus({ case: "copied" });
    } catch (error) {
      setStatus({
        case: "failed",
        message:
          error instanceof Error ? error.message : t("inspector.promptFailed"),
      });
    } finally {
      setPending(false);
    }
  }

  return (
    <div className="inspector-prompt">
      <IconButton
        icon={status.case === "copied" ? Check : ClipboardCopy}
        label={label}
        variant="secondary"
        type="button"
        disabled={pending}
        onClick={() => void copyPrompt()}
      />
      <p className="inspector-prompt-status" aria-live="polite">
        {status.case === "copied" && t("inspector.promptCopied")}
        {status.case === "failed" && status.message}
      </p>
    </div>
  );
}
