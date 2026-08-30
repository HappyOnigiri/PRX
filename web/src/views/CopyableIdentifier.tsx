import { Check, Copy } from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { IconButton } from "./IconButton";

type CopyStatus = "copied" | "failed";

export function CopyableIdentifier({
  label,
  value,
}: {
  label: string;
  value: string;
}) {
  const { t } = useTranslation();
  const [status, setStatus] = useState<CopyStatus>();

  async function copyIdentifier() {
    try {
      await navigator.clipboard.writeText(value);
      setStatus("copied");
    } catch {
      setStatus("failed");
    }
    window.setTimeout(() => {
      setStatus(undefined);
    }, 1600);
  }

  const copyLabel = t("common.copyIdentifier", { label });
  const isCopied = status === "copied";
  const Icon = isCopied ? Check : Copy;
  return (
    <span className="copyable-identifier">
      <span className="copyable-identifier-label">{label}</span>
      <code className="copyable-identifier-value" title={value}>
        {value}
      </code>
      <IconButton
        icon={Icon}
        label={copyLabel}
        variant="quiet"
        size="compact"
        iconOnly
        className={`copyable-identifier-button${isCopied ? " is-copied" : ""}`}
        aria-label={isCopied ? t("common.copied") : copyLabel}
        title={copyLabel}
        onClick={() => void copyIdentifier()}
        iconProps={{ "data-icon": isCopied ? "check" : "copy" }}
      />
      <span className="copyable-identifier-status" aria-live="polite">
        {status === "failed" && t("common.copyFailed")}
      </span>
    </span>
  );
}
