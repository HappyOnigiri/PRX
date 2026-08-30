import { AlertTriangle, Check, X } from "lucide-react";
import { useEffect, useId, useRef } from "react";
import { useTranslation } from "react-i18next";
import { formatError } from "../i18n/domain";
import { IconButton } from "./IconButton";

interface ConfirmationDialogProps {
  title: string;
  description: string;
  confirmLabel: string;
  danger?: boolean;
  pending: boolean;
  error: Error | null;
  onCancel: () => void;
  onConfirm: () => void;
}

export function ConfirmationDialog({
  title,
  description,
  confirmLabel,
  danger = false,
  pending,
  error,
  onCancel,
  onConfirm,
}: ConfirmationDialogProps) {
  const { t } = useTranslation();
  const titleId = useId();
  const descriptionId = useId();
  const cancelRef = useRef<HTMLButtonElement>(null);
  const dialogRef = useRef<HTMLElement>(null);

  useEffect(() => {
    const origin = document.activeElement;
    cancelRef.current?.focus();
    return () => {
      window.setTimeout(() => {
        if (origin instanceof HTMLElement && origin.isConnected) origin.focus();
      });
    };
  }, []);

  return (
    <div
      className="scrim confirmation-scrim"
      role="presentation"
      onKeyDown={(event) => {
        if (event.key === "Escape" && !pending) onCancel();
        if (event.key !== "Tab") return;
        const focusable = Array.from(
          dialogRef.current?.querySelectorAll<HTMLButtonElement>(
            "button:not(:disabled)",
          ) ?? [],
        );
        if (focusable.length === 0) {
          event.preventDefault();
          return;
        }
        const first = focusable[0];
        const last = focusable.at(-1);
        if (!first || !last) return;
        if (event.shiftKey && document.activeElement === first) {
          event.preventDefault();
          last.focus();
        } else if (!event.shiftKey && document.activeElement === last) {
          event.preventDefault();
          first.focus();
        }
      }}
    >
      <section
        ref={dialogRef}
        className="dialog confirmation-dialog"
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        aria-describedby={descriptionId}
      >
        <header>
          <span
            className={
              danger ? "confirmation-icon danger" : "confirmation-icon"
            }
          >
            <AlertTriangle aria-hidden="true" focusable="false" size={20} />
          </span>
          <div>
            <h2 id={titleId}>{title}</h2>
            <p id={descriptionId}>{description}</p>
          </div>
        </header>
        {error && (
          <p className="form-error" role="alert">
            {formatError(error, t)}
          </p>
        )}
        <footer>
          <IconButton
            ref={cancelRef}
            icon={X}
            label={t("common.cancel")}
            variant="secondary"
            disabled={pending}
            onClick={onCancel}
          />
          <IconButton
            icon={Check}
            label={confirmLabel}
            variant={danger ? "danger" : "primary"}
            disabled={pending}
            onClick={onConfirm}
          />
        </footer>
      </section>
    </div>
  );
}
