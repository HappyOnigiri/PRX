import { useTranslation } from "react-i18next";
import { setDisplayLanguage } from "../i18n";
import { supportedLanguages, type SupportedLanguage } from "../i18n/settings";

export function LanguageSelector() {
  const { t, i18n } = useTranslation();
  return (
    <label className="language-setting">
      <span>{t("language.label")}</span>
      <select
        aria-label={t("language.label")}
        value={i18n.resolvedLanguage ?? "en"}
        onChange={(event) =>
          void setDisplayLanguage(event.target.value as SupportedLanguage)
        }
      >
        {supportedLanguages.map((language) => (
          <option value={language} key={language}>
            {t(`language.${language}`)}
          </option>
        ))}
      </select>
    </label>
  );
}
