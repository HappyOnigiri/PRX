import { Check, X } from "lucide-react";
import { useRef, useState, type KeyboardEvent, type ReactNode } from "react";
import { useTranslation } from "react-i18next";
import { setDisplayLanguage } from "../i18n";
import {
  readThemePreference,
  supportedLanguages,
  themePreferences,
  type SupportedLanguage,
  type ThemePreference,
} from "../i18n/settings";
import { setDisplayTheme } from "../theme";
import { DebugSettingsPanel } from "./DebugSettingsPanel";
import { IconButton } from "./IconButton";
import { LicensesSettingsPanel } from "./LicensesSettingsPanel";
import { ServerSettingsPanel } from "./ServerSettingsPanel";

const settingsTabs = ["server", "display", "debug", "licenses"] as const;
type SettingsTab = (typeof settingsTabs)[number];

export function SettingsDialog({ onClose }: { onClose: () => void }) {
  const { t } = useTranslation();
  const [activeTab, setActiveTab] = useState<SettingsTab>("server");
  const tabRefs = useRef<(HTMLButtonElement | null)[]>([]);

  function selectTab(index: number) {
    const tab = settingsTabs[index];
    if (!tab) return;
    setActiveTab(tab);
    tabRefs.current[index]?.focus();
  }

  function handleTabKey(event: KeyboardEvent<HTMLButtonElement>) {
    const current = settingsTabs.indexOf(activeTab);
    let next: number | undefined;
    if (event.key === "ArrowRight") next = (current + 1) % settingsTabs.length;
    if (event.key === "ArrowLeft")
      next = (current - 1 + settingsTabs.length) % settingsTabs.length;
    if (event.key === "Home") next = 0;
    if (event.key === "End") next = settingsTabs.length - 1;
    if (next === undefined) return;
    event.preventDefault();
    selectTab(next);
  }

  return (
    <div className="scrim" role="presentation">
      <section
        className="dialog settings-dialog"
        role="dialog"
        aria-modal="true"
        aria-labelledby="settings-dialog-title"
      >
        <header className="settings-dialog-head">
          <h2 id="settings-dialog-title">{t("settings.title")}</h2>
          <IconButton
            icon={X}
            label={t("common.close")}
            variant="secondary"
            iconOnly
            type="button"
            onClick={onClose}
          />
        </header>
        <div className="settings-tabs" role="tablist">
          {settingsTabs.map((tab, index) => (
            <button
              aria-controls={`settings-panel-${tab}`}
              aria-selected={activeTab === tab}
              className="settings-tab"
              id={`settings-tab-${tab}`}
              key={tab}
              onClick={() => {
                setActiveTab(tab);
              }}
              onKeyDown={handleTabKey}
              ref={(element) => {
                tabRefs.current[index] = element;
              }}
              role="tab"
              tabIndex={activeTab === tab ? 0 : -1}
              type="button"
            >
              {t(`settings.tabs.${tab}`)}
            </button>
          ))}
        </div>
        <SettingsPanel active={activeTab === "server"} tab="server">
          <ServerSettingsPanel />
        </SettingsPanel>
        <SettingsPanel active={activeTab === "display"} tab="display">
          <DisplaySettingsPanel />
        </SettingsPanel>
        <SettingsPanel active={activeTab === "debug"} tab="debug">
          {/* Unlike the other panels, this one mounts only while it is active:
              collecting a report reads the database and the configuration file,
              and opening the dialog must not do that on its own. */}
          {activeTab === "debug" && <DebugSettingsPanel />}
        </SettingsPanel>
        <SettingsPanel active={activeTab === "licenses"} tab="licenses">
          <LicensesSettingsPanel />
        </SettingsPanel>
        <footer>
          <IconButton
            icon={Check}
            label={t("common.done")}
            variant="secondary"
            type="button"
            onClick={onClose}
          />
        </footer>
      </section>
    </div>
  );
}

function SettingsPanel({
  active,
  children,
  tab,
}: {
  active: boolean;
  children: ReactNode;
  tab: SettingsTab;
}) {
  return (
    <div
      aria-labelledby={`settings-tab-${tab}`}
      className="settings-tab-panel"
      hidden={!active}
      id={`settings-panel-${tab}`}
      role="tabpanel"
      tabIndex={0}
    >
      {children}
    </div>
  );
}

function DisplaySettingsPanel() {
  const { t, i18n } = useTranslation();
  const [theme, setTheme] = useState(readThemePreference);
  return (
    <>
      <p className="dialog-lead">{t("settings.display.description")}</p>
      <div className="settings-display-list">
        <label className="settings-display-row">
          <span className="settings-display-copy">
            <strong>{t("settings.display.language.label")}</strong>
            <small>{t("settings.display.language.description")}</small>
          </span>
          <select
            aria-label={t("settings.display.language.label")}
            value={i18n.resolvedLanguage ?? "en"}
            onChange={(event) => {
              void setDisplayLanguage(event.target.value as SupportedLanguage);
            }}
          >
            {supportedLanguages.map((language) => (
              <option value={language} key={language}>
                {t(`settings.display.language.options.${language}`)}
              </option>
            ))}
          </select>
        </label>
        <label className="settings-display-row">
          <span className="settings-display-copy">
            <strong>{t("settings.display.theme.label")}</strong>
            <small>{t("settings.display.theme.description")}</small>
          </span>
          <select
            aria-label={t("settings.display.theme.label")}
            value={theme}
            onChange={(event) => {
              const preference = event.target.value as ThemePreference;
              setTheme(preference);
              setDisplayTheme(preference);
            }}
          >
            {themePreferences.map((preference) => (
              <option value={preference} key={preference}>
                {t(`settings.display.theme.options.${preference}`)}
              </option>
            ))}
          </select>
        </label>
      </div>
    </>
  );
}
