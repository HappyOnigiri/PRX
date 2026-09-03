import { useNavigate } from "@tanstack/react-router";
import { Plus, X } from "lucide-react";
import { type SyntheticEvent } from "react";
import { useTranslation } from "react-i18next";
import { mutations } from "../api";
import { formValue } from "../form";
import { useDomainMutation } from "../hooks";
import { formatError } from "../i18n/domain";
import { IconButton } from "./IconButton";

export function ProjectCreateDialog({ onClose }: { onClose: () => void }) {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const createProject = useDomainMutation(mutations.createProject);

  async function submit(event: SyntheticEvent<HTMLFormElement>) {
    event.preventDefault();
    const data = new FormData(event.currentTarget);
    let response;
    try {
      response = await createProject.mutateAsync({
        slug: formValue(data, "slug"),
        title: formValue(data, "title"),
        description: formValue(data, "description"),
      });
    } catch {
      return;
    }
    onClose();
    if (response.project)
      await navigate({
        to: "/projects/$projectId",
        params: { projectId: response.project.id },
      });
  }

  return (
    <div className="scrim" role="presentation">
      <form
        className="dialog"
        onSubmit={submit}
        aria-label={t("projectCreate.formLabel")}
      >
        <header>
          <h2>{t("projectCreate.title")}</h2>
        </header>
        <label>
          {t("common.slug")}
          <input
            name="slug"
            required
            pattern="[a-z0-9]+(?:-[a-z0-9]+)*"
            placeholder={t("projectCreate.slugPlaceholder")}
          />
        </label>
        <label>
          {t("common.title")}
          <input
            name="title"
            required
            placeholder={t("projectCreate.titlePlaceholder")}
          />
        </label>
        <label>
          {t("common.description")}
          <textarea
            name="description"
            placeholder={t("projectCreate.descriptionPlaceholder")}
          />
        </label>
        {createProject.error && (
          <p className="form-error">{formatError(createProject.error, t)}</p>
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
            label={t("projectCreate.submit")}
            variant="primary"
            type="submit"
            disabled={createProject.isPending}
          />
        </footer>
      </form>
    </div>
  );
}
