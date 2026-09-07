import { RotateCcw, Save } from "lucide-react";
import { useRef, useState, type SyntheticEvent } from "react";
import { useTranslation } from "react-i18next";
import { promptMutations, type PromptTemplateSettings } from "../api";
import { usePromptTemplates, usePromptTemplatesMutation } from "../hooks";
import { IconButton } from "./IconButton";

interface TemplateDraft {
  design: string;
  implementation: string;
  batch: string;
}

export function PromptSettingsPanel() {
  const { t } = useTranslation();
  const templates = usePromptTemplates();
  const update = usePromptTemplatesMutation(promptMutations.updateTemplates);
  const [draft, setDraft] = useState<TemplateDraft>();
  const [saved, setSaved] = useState(false);
  // キー入力を数え、実行中の保存が、送ったテキストと画面上のテキストが同じか
  // 判定できるようにする。
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
    batch: templates.data.batch,
  };
  const builtIn = {
    design: templates.data.builtIn.design,
    implementation: templates.data.builtIn.implementation,
    batch: templates.data.builtIn.batch,
  };

  // テンプレートはすべて 1 リクエストで送り、設定の書き込みで片方だけ更新され
  // 片方が古いまま残ることがないようにする。
  async function submit(event: SyntheticEvent<HTMLFormElement>) {
    event.preventDefault();
    setSaved(false);
    const submitted = edits.current;
    const result = await update.mutateAsync(current);
    // リクエスト中に打たれたテキストはレスポンスより新しい。サーバーのコピーを
    // 採用するとその入力を黙って巻き戻したうえ、ユーザーにもう見えないテキスト
    // について成功を報告してしまう。
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
        <TemplateFields
          current={current}
          settings={templates.data}
          onEdit={edit}
        />
        <div className="settings-form-actions">
          <IconButton
            icon={RotateCcw}
            label={t("promptSettings.restoreDefaults")}
            variant="secondary"
            type="button"
            disabled={update.isPending}
            onClick={() => {
              // 空のテンプレートならサーバーが自前で既定値に戻すが、エディタは
              // これから保存する内容を見せるものなので、書き込みを待たず組み
              // 込みのテキストをフィールドに入れる。
              edit(builtIn);
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

// 語彙はサーバー由来なので、サーバーが拒む placeholder をヒントが案内すること
// はない。各フィールドは自分の分だけを列挙する。
// docs/design/agent-prompts.md を参照。
function TemplateFields({
  current,
  settings,
  onEdit,
}: {
  current: TemplateDraft;
  settings: PromptTemplateSettings;
  onEdit: (next: TemplateDraft) => void;
}) {
  const { t } = useTranslation();
  return (
    <>
      <TemplateField
        hint={t("promptSettings.designHint")}
        label={t("promptSettings.design")}
        value={current.design}
        onChange={(design) => {
          onEdit({ ...current, design });
        }}
      />
      <TemplateField
        hint={t("promptSettings.implementationHint")}
        label={t("promptSettings.implementation")}
        value={current.implementation}
        onChange={(implementation) => {
          onEdit({ ...current, implementation });
        }}
      />
      <small>
        {t("promptSettings.placeholders", {
          list: placeholderList(settings.supportedPlaceholders),
          required: `{{${settings.requiredPlaceholder}}}`,
        })}
      </small>
      <TemplateField
        hint={t("promptSettings.batchHint")}
        label={t("promptSettings.batch")}
        value={current.batch}
        onChange={(batch) => {
          onEdit({ ...current, batch });
        }}
      />
      <small>
        {t("promptSettings.batchPlaceholders", {
          list: placeholderList(settings.batchSupportedPlaceholders),
          required: `{{${settings.batchRequiredPlaceholder}}}`,
        })}
      </small>
    </>
  );
}

// リストも必須名も補間値なので波括弧が残る。波括弧を含む翻訳文にすると、それ
// 自体が補間で消えてしまう。
function placeholderList(names: string[]): string {
  return names.map((name) => `{{${name}}}`).join(", ");
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
