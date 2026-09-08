import { RotateCcw } from "lucide-react";
import { useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { promptMutations, type PromptTemplateSettings } from "../api";
import { usePromptTemplates, usePromptTemplatesMutation } from "../hooks";
import { IconButton } from "./IconButton";
import { useRegisterSettingsSection } from "./settingsSections";

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
  // キー入力を数え、実行中の保存が、送ったテキストと画面上のテキストが同じか
  // 判定できるようにする。
  const edits = useRef(0);

  const saved = templates.data ? templatesOf(templates.data) : undefined;
  const current = draft ?? saved;

  function edit(next: TemplateDraft) {
    edits.current += 1;
    setDraft(next);
  }

  // テンプレートはすべて 1 リクエストで送り、設定の書き込みで片方だけ更新され
  // 片方が古いまま残ることがないようにする。
  async function save() {
    if (!current) return;
    const submitted = edits.current;
    const result = await update.mutateAsync(current);
    // リクエスト中に打たれたテキストはレスポンスより新しい。サーバーのコピーを
    // 採用するとその入力を黙って巻き戻したうえ、ユーザーにもう見えないテキスト
    // について成功を報告してしまう。
    if (edits.current !== submitted) return;
    if (result.templates) setDraft(result.templates);
  }

  useRegisterSettingsSection("prompts", {
    dirty: Boolean(current && saved && !sameTemplates(current, saved)),
    invalid: false,
    save,
  });

  if (templates.isPending) {
    return (
      <p className="settings-panel-state">{t("promptSettings.loading")}</p>
    );
  }
  if (!templates.data) {
    return <p className="form-error">{templates.error.message}</p>;
  }
  const editable = draft ?? templatesOf(templates.data);

  return (
    <>
      <p className="dialog-lead">{t("promptSettings.description")}</p>
      <div className="settings-form">
        <TemplateFields
          current={editable}
          settings={templates.data}
          onEdit={edit}
        />
        <div className="settings-form-actions">
          <IconButton
            icon={RotateCcw}
            label={t("promptSettings.restoreDefaults")}
            variant="secondary"
            disabled={update.isPending}
            onClick={() => {
              // 空のテンプレートならサーバーが自前で既定値に戻すが、エディタは
              // これから保存する内容を見せるものなので、書き込みを待たず組み
              // 込みのテキストをフィールドに入れる。
              edit({
                design: templates.data.builtIn.design,
                implementation: templates.data.builtIn.implementation,
                batch: templates.data.builtIn.batch,
              });
            }}
          />
        </div>
      </div>
      {update.error && <p className="form-error">{update.error.message}</p>}
    </>
  );
}

function templatesOf(templates: PromptTemplateSettings): TemplateDraft {
  return {
    design: templates.design,
    implementation: templates.implementation,
    batch: templates.batch,
  };
}

function sameTemplates(left: TemplateDraft, right: TemplateDraft): boolean {
  return (
    left.design === right.design &&
    left.implementation === right.implementation &&
    left.batch === right.batch
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
