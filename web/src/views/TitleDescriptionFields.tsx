import { useTranslation } from "react-i18next";

// タイトルと説明の組は project と feature の編集で同一なので、両方の下書きを
// 扱える 1 つのフィールド群にまとめる。
export function TitleDescriptionFields<
  T extends { title: string; description: string },
>({ draft, onChange }: { draft: T; onChange: (next: T) => void }) {
  const { t } = useTranslation();
  return (
    <>
      <label>
        {t("common.title")}
        <input
          name="title"
          required
          value={draft.title}
          onChange={(event) => {
            onChange({ ...draft, title: event.target.value });
          }}
        />
      </label>
      <label>
        {t("common.description")}
        <textarea
          name="description"
          value={draft.description}
          onChange={(event) => {
            onChange({ ...draft, description: event.target.value });
          }}
        />
      </label>
    </>
  );
}
