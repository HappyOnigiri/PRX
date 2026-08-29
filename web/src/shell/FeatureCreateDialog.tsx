import type { SyntheticEvent } from "react";
import { useNavigate } from "@tanstack/react-router";
import { useTranslation } from "react-i18next";
import { mutations } from "../api";
import { formValue } from "../form";
import { useDomainMutation } from "../hooks";
import { formatError } from "../i18n/domain";

interface FeatureCreateDialogProps {
  onClose: () => void;
}

export function FeatureCreateDialog({ onClose }: FeatureCreateDialogProps) {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const createFeature = useDomainMutation(mutations.createFeature);

  async function submit(event: SyntheticEvent<HTMLFormElement>) {
    event.preventDefault();
    const data = new FormData(event.currentTarget);
    const response = await createFeature.mutateAsync({
      slug: formValue(data, "slug"),
      title: formValue(data, "title"),
      description: formValue(data, "description"),
    });
    onClose();
    if (response.feature)
      await navigate({
        to: "/features/$featureId",
        params: { featureId: response.feature.id },
      });
  }

  return (
    <div className="scrim" role="presentation">
      <form
        className="dialog"
        onSubmit={submit}
        aria-label={t("featureCreate.formLabel")}
      >
        <header>
          <h2>{t("featureCreate.title")}</h2>
        </header>
        <label>
          {t("common.slug")}
          <input
            name="slug"
            required
            pattern="[a-z0-9]+(?:-[a-z0-9]+)*"
            placeholder={t("featureCreate.slugPlaceholder")}
          />
        </label>
        <label>
          {t("common.title")}
          <input
            name="title"
            required
            placeholder={t("featureCreate.titlePlaceholder")}
          />
        </label>
        <label>
          {t("common.description")}
          <textarea
            name="description"
            placeholder={t("featureCreate.descriptionPlaceholder")}
          />
        </label>
        {createFeature.error && (
          <p className="form-error">{formatError(createFeature.error, t)}</p>
        )}
        <footer>
          <button type="button" className="secondary" onClick={onClose}>
            {t("common.cancel")}
          </button>
          <button disabled={createFeature.isPending}>
            {t("featureCreate.submit")}
          </button>
        </footer>
      </form>
    </div>
  );
}
