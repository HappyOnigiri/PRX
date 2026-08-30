import { Save, X } from "lucide-react";
import type { SyntheticEvent } from "react";
import { useTranslation } from "react-i18next";
import { mutations } from "../api";
import { formValue } from "../form";
import { FeatureStatus, type Feature } from "../gen/prx/v1/prx_pb";
import { useDomainMutation } from "../hooks";
import { featureStatusLabel } from "../i18n/domain";
import { IconButton } from "./IconButton";
import { MutationError } from "./MutationError";

interface EditFeatureDialogProps {
  feature: Feature;
  onClose: () => void;
}

export function EditFeatureDialog({
  feature,
  onClose,
}: EditFeatureDialogProps) {
  const { t } = useTranslation();
  const updateFeature = useDomainMutation(mutations.updateFeature);

  async function submitFeature(event: SyntheticEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = new FormData(event.currentTarget);
    try {
      await updateFeature.mutateAsync({
        id: feature.id,
        slug: formValue(form, "slug"),
        title: formValue(form, "title"),
        description: formValue(form, "description"),
        status: Number(form.get("status")),
      });
    } catch {
      return;
    }
    onClose();
  }

  return (
    <div className="scrim">
      <form
        className="dialog"
        onSubmit={submitFeature}
        aria-label={t("featureEdit.formLabel")}
      >
        <header>
          <h2>{t("featureEdit.title")}</h2>
        </header>
        <label>
          {t("common.slug")}
          <input name="slug" required defaultValue={feature.slug} />
        </label>
        <label>
          {t("common.title")}
          <input name="title" required defaultValue={feature.title} />
        </label>
        <label>
          {t("common.description")}
          <textarea name="description" defaultValue={feature.description} />
        </label>
        <label>
          {t("common.status")}
          <select name="status" defaultValue={feature.status}>
            {[
              FeatureStatus.ACTIVE,
              FeatureStatus.PAUSED,
              FeatureStatus.COMPLETED,
              FeatureStatus.CANCELLED,
            ].map((status) => (
              <option value={status} key={status}>
                {featureStatusLabel(status, t)}
              </option>
            ))}
          </select>
        </label>
        <MutationError error={updateFeature.error} />
        <footer>
          <IconButton
            icon={X}
            label={t("common.cancel")}
            variant="secondary"
            onClick={onClose}
          />
          <IconButton
            icon={Save}
            label={t("featureEdit.submit")}
            variant="primary"
            type="submit"
            disabled={updateFeature.isPending}
          />
        </footer>
      </form>
    </div>
  );
}
