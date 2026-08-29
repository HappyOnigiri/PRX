import { useState } from "react";
import { useTranslation } from "react-i18next";
import {
  readThemePreference,
  themePreferences,
  type ThemePreference,
} from "../i18n/settings";
import { setDisplayTheme } from "../theme";

export function ThemeSelector() {
  const { t } = useTranslation();
  const [theme, setTheme] = useState(readThemePreference);
  return (
    <label className="theme-setting">
      <span>{t("theme.label")}</span>
      <select
        aria-label={t("theme.label")}
        value={theme}
        onChange={(event) => {
          const preference = event.target.value as ThemePreference;
          setTheme(preference);
          setDisplayTheme(preference);
        }}
      >
        {themePreferences.map((preference) => (
          <option value={preference} key={preference}>
            {t(`theme.${preference}`)}
          </option>
        ))}
      </select>
    </label>
  );
}
