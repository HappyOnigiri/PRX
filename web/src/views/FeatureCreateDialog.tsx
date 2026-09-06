import { useNavigate } from "@tanstack/react-router";
import { Plus, X } from "lucide-react";
import { type SyntheticEvent } from "react";
import { useTranslation } from "react-i18next";
import { mutations } from "../api";
import { formValue } from "../form";
import { useDomainMutation } from "../hooks";
import { formatError } from "../i18n/domain";
import { IconButton } from "./IconButton";

// A feature always belongs to a project, so it is created from the project's
// own page and the membership needs no field: the page the caller is on names
// it.
export function FeatureCreateDialog({
  projectId,
  onClose,
}: {
  projectId: string;
  onClose: () => void;
}) {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const createFeature = useDomainMutation(mutations.createFeature);

  async function submit(event: SyntheticEvent<HTMLFormElement>) {
    event.preventDefault();
    const data = new FormData(event.currentTarget);
    let response;
    try {
      response = await createFeature.mutateAsync({
        title: formValue(data, "title"),
        description: formValue(data, "description"),
        projectId,
      });
    } catch {
      return;
    }
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
          <IconButton
            icon={X}
            label={t("common.cancel")}
            variant="secondary"
            onClick={onClose}
          />
          <IconButton
            icon={Plus}
            label={t("featureCreate.submit")}
            variant="primary"
            type="submit"
            disabled={createFeature.isPending}
          />
        </footer>
      </form>
    </div>
  );
}
