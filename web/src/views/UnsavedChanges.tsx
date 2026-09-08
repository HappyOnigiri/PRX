import { Save } from "lucide-react";
import { useTranslation } from "react-i18next";
import { ConfirmationDialog } from "./ConfirmationDialog";
import { IconButton, type IconButtonProps } from "./IconButton";

// 編集面はどれも「保存は 1 つ、閉じる前に破棄を確認する」で揃える。
// docs/design/webui.md を参照。

export function SaveButton({
  dirty,
  label,
  pending,
  ...buttonProps
}: {
  dirty: boolean;
  label?: string;
  pending: boolean;
} & Omit<IconButtonProps, "icon" | "label">) {
  const { t } = useTranslation();
  return (
    <IconButton
      {...buttonProps}
      icon={Save}
      label={label ?? t("common.save")}
      // 保存できるものがあるときだけ緑を点け、押せる状態を色でも示す。
      variant={dirty ? "affirm" : "secondary"}
      disabled={!dirty || pending}
    />
  );
}

export function DiscardChangesDialog({
  onCancel,
  onConfirm,
}: {
  onCancel: () => void;
  onConfirm: () => void;
}) {
  const { t } = useTranslation();
  return (
    <ConfirmationDialog
      title={t("common.discard.title")}
      description={t("common.discard.description")}
      confirmLabel={t("common.discard.confirm")}
      danger
      pending={false}
      error={null}
      onCancel={onCancel}
      onConfirm={onConfirm}
    />
  );
}

export function SaveStatus({
  pending,
  saved,
}: {
  pending: boolean;
  saved: boolean;
}) {
  const { t } = useTranslation();
  return (
    <p className="save-status" aria-live="polite">
      {pending && t("common.saving")}
      {!pending && saved && t("common.saved")}
    </p>
  );
}
