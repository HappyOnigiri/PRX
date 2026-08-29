import { Copy } from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";

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
  return (
    <span className="copyable-identifier">
      <span className="copyable-identifier-label">{label}</span>
      <code className="copyable-identifier-value" title={value}>
        {value}
      </code>
      <button
        type="button"
        className="copyable-identifier-button"
        aria-label={copyLabel}
        title={copyLabel}
        onClick={() => void copyIdentifier()}
      >
        <Copy
          aria-hidden="true"
          focusable="false"
          size={14}
          strokeWidth={1.35}
        />
      </button>
      <span className="copyable-identifier-status" aria-live="polite">
        {status === "copied" && t("common.copied")}
        {status === "failed" && t("common.copyFailed")}
      </span>
    </span>
  );
}
