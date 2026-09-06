import { Check, X } from "lucide-react";
import { useState, type ReactNode } from "react";
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
import { PromptSettingsPanel } from "./PromptSettingsPanel";
import { ServerSettingsPanel } from "./ServerSettingsPanel";
import { TabList, TabPanel } from "./TabList";

const settingsTabs = [
  "server",
  "prompts",
  "display",
  "debug",
  "licenses",
] as const;
type SettingsTab = (typeof settingsTabs)[number];

export function SettingsDialog({ onClose }: { onClose: () => void }) {
  const { t } = useTranslation();
  const [activeTab, setActiveTab] = useState<SettingsTab>("server");
  const [promptsMounted, setPromptsMounted] = useState(false);

  function openTab(tab: SettingsTab) {
    setActiveTab(tab);
    // The prompt panel holds unsaved edits in component state, so once it has
    // been opened it stays mounted until the dialog closes: leaving the tab and
    // coming back must not discard what the user typed without warning.
    if (tab === "prompts") setPromptsMounted(true);
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
        <TabList
          tabs={settingsTabs.map((tab) => ({
            id: tab,
            label: t(`settings.tabs.${tab}`),
          }))}
          active={activeTab}
          onSelect={openTab}
          idPrefix="settings"
          className="settings-tabs"
          tabClassName="settings-tab"
        />
        <SettingsPanel active={activeTab === "server"} tab="server">
          <ServerSettingsPanel />
        </SettingsPanel>
        <SettingsPanel active={activeTab === "prompts"} tab="prompts">
          {/* Like the debug panel, this one reads the configuration file, so it
              mounts only once the tab is opened. Unlike it, it stays mounted
              afterwards so unsaved edits survive a trip to another tab. */}
          {promptsMounted && <PromptSettingsPanel />}
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
    <TabPanel
      active={active}
      className="settings-tab-panel"
      idPrefix="settings"
      tab={tab}
    >
      {children}
    </TabPanel>
  );
}

function DisplaySettingsPanel() {
  const { t, i18n } = useTranslation();
  const [theme, setTheme] = useState(readThemePreference);
  return (
    <>
      <div className="settings-display-list">
        <label className="settings-display-row">
          <span className="settings-display-copy">
            <strong>{t("settings.display.language.label")}</strong>
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
