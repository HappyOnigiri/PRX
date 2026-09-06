import { RotateCcw, Save } from "lucide-react";
import { useRef, useState, type SyntheticEvent } from "react";
import { useTranslation } from "react-i18next";
import { promptMutations } from "../api";
import { usePromptTemplates, usePromptTemplatesMutation } from "../hooks";
import { IconButton } from "./IconButton";

interface TemplateDraft {
  design: string;
  implementation: string;
}

export function PromptSettingsPanel() {
  const { t } = useTranslation();
  const templates = usePromptTemplates();
  const update = usePromptTemplatesMutation(promptMutations.updateTemplates);
  const [draft, setDraft] = useState<TemplateDraft>();
  const [saved, setSaved] = useState(false);
  // Counts keystrokes so a save that is still in flight can tell whether the
  // text it sent is still the text on screen.
  const edits = useRef(0);

  function edit(next: TemplateDraft) {
    edits.current += 1;
    setSaved(false);
    setDraft(next);
  }

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

  // The vocabulary comes from the server so the hint can never advertise a
  // placeholder the server would reject. Both are interpolation values so the
  // braces survive: a translation containing them would itself be interpolated
  // away.
  const placeholderList = templates.data.supportedPlaceholders
    .map((name) => `{{${name}}}`)
    .join(", ");
  const requiredPlaceholder = `{{${templates.data.requiredPlaceholder}}}`;

  // Both templates travel in one request so a configuration write never leaves
  // one of them updated and the other stale.
  async function submit(event: SyntheticEvent<HTMLFormElement>) {
    event.preventDefault();
    setSaved(false);
    const submitted = edits.current;
    const result = await update.mutateAsync(current);
    // Text typed while the request was in flight is newer than the response, so
    // adopting the server's copy would silently revert those keystrokes and
    // then report success for text the user no longer sees.
    if (edits.current !== submitted) return;
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
            edit({ ...current, design });
          }}
        />
        <TemplateField
          hint={t("promptSettings.implementationHint")}
          label={t("promptSettings.implementation")}
          value={current.implementation}
          onChange={(implementation) => {
            edit({ ...current, implementation });
          }}
        />
        <small>
          {t("promptSettings.placeholders", {
            list: placeholderList,
            required: requiredPlaceholder,
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
              edit({ design: "", implementation: "" });
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
