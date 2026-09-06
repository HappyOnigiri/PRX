import { RotateCcw, Save } from "lucide-react";
import { useState, type SyntheticEvent } from "react";
import { useTranslation } from "react-i18next";
import { promptMutations } from "../api";
import { usePromptTemplates, usePromptTemplatesMutation } from "../hooks";
import { IconButton } from "./IconButton";

interface TemplateDraft {
  design: string;
  implementation: string;
}

// The vocabulary the server accepts. It is listed through an interpolation
// value so the braces survive: a translation containing them would itself be
// interpolated away.
const placeholderNames = [
  "task_id",
  "feature_id",
  "task_title",
  "task_scope",
  "task_kind",
] as const;
const placeholderList = placeholderNames
  .map((name) => `{{${name}}}`)
  .join(", ");

export function PromptSettingsPanel() {
  const { t } = useTranslation();
  const templates = usePromptTemplates();
  const update = usePromptTemplatesMutation(promptMutations.updateTemplates);
  const [draft, setDraft] = useState<TemplateDraft>();
  const [saved, setSaved] = useState(false);

  if (templates.isPending) {
    return (
      <p className="settings-panel-state">{t("promptSettings.loading")}</p>
    );
  }
  if (!templates.data) {
    return <p className="form-error">{templates.error.message}</p>;
  }

  const current = draft ?? {
    design: templates.data.design,
    implementation: templates.data.implementation,
  };

  // Both templates travel in one request so a configuration write never leaves
  // one of them updated and the other stale.
  async function submit(event: SyntheticEvent<HTMLFormElement>) {
    event.preventDefault();
    setSaved(false);
    const result = await update.mutateAsync(current);
    if (result.templates) setDraft(result.templates);
    setSaved(true);
  }

  return (
    <>
      <p className="dialog-lead">{t("promptSettings.description")}</p>
      <form
        className="settings-form"
        onSubmit={(event) => {
          void submit(event).catch(() => undefined);
        }}
      >
        <TemplateField
          hint={t("promptSettings.designHint")}
          label={t("promptSettings.design")}
          value={current.design}
          onChange={(design) => {
            setSaved(false);
            setDraft({ ...current, design });
          }}
        />
        <TemplateField
          hint={t("promptSettings.implementationHint")}
          label={t("promptSettings.implementation")}
          value={current.implementation}
          onChange={(implementation) => {
            setSaved(false);
            setDraft({ ...current, implementation });
          }}
        />
        <small>
          {t("promptSettings.placeholders", {
            list: placeholderList,
            required: "{{task_id}}",
          })}
        </small>
        <div className="settings-form-actions">
          <IconButton
            icon={RotateCcw}
            label={t("promptSettings.restoreDefaults")}
            variant="secondary"
            type="button"
            disabled={update.isPending}
            onClick={() => {
              setSaved(false);
              setDraft({ design: "", implementation: "" });
            }}
          />
          <IconButton
            icon={Save}
            label={t("common.save")}
            variant="primary"
            type="submit"
            disabled={update.isPending}
          />
        </div>
      </form>
      {update.error && <p className="form-error">{update.error.message}</p>}
      <p className="settings-form-status" aria-live="polite">
        {saved && !update.error && t("promptSettings.saved")}
      </p>
    </>
  );
}

function TemplateField({
  hint,
  label,
  onChange,
  value,
}: {
  hint: string;
  label: string;
  onChange: (value: string) => void;
  value: string;
}) {
  return (
    <label className="settings-prompt-field">
      {label}
      <textarea
        rows={12}
        value={value}
        onChange={(event) => {
          onChange(event.target.value);
        }}
      />
      <small>{hint}</small>
    </label>
  );
}
