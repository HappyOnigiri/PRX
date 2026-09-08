import { X } from "lucide-react";
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
import {
  useRegisterSettingsSection,
  useSettingsSections,
} from "./settingsSections";
import { TabList, TabPanel } from "./TabList";
import { DiscardChangesDialog, SaveButton, SaveStatus } from "./UnsavedChanges";

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
  const [discarding, setDiscarding] = useState(false);
  const [saving, setSaving] = useState(false);
  const [saved, setSaved] = useState(false);
  const sections = useSettingsSections();

  function openTab(tab: SettingsTab) {
    setActiveTab(tab);
    // プロンプトのパネルは未保存の編集をコンポーネント state に持つので、一度
    // 開いたらダイアログを閉じるまでマウントしたままにする。タブを離れて戻った
    // だけで入力内容を警告なく捨ててはならないため。
    if (tab === "prompts") setPromptsMounted(true);
  }

  // 保存はタブをまたいで 1 つなので、押した時点の下書きをすべて書き込む。
  // 失敗した分はそれぞれのパネルが理由を出す。
  async function save() {
    setSaving(true);
    setSaved(false);
    try {
      await sections.saveAll();
      setSaved(true);
    } catch {
      setSaved(false);
    } finally {
      setSaving(false);
    }
  }

  function requestClose() {
    if (sections.dirty) setDiscarding(true);
    else onClose();
  }

  return (
    <>
      <div
        className="scrim"
        role="presentation"
        aria-hidden={discarding ? true : undefined}
        inert={discarding}
      >
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
              onClick={requestClose}
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
          {sections.provider(
            <>
              <SettingsPanel active={activeTab === "server"} tab="server">
                <ServerSettingsPanel />
              </SettingsPanel>
              <SettingsPanel active={activeTab === "prompts"} tab="prompts">
                {/* debug パネルと同じく設定ファイルを読むので、タブを開いたときだけ
                    マウントする。ただし未保存の編集が他タブへの移動で消えないよう、
                    以後はマウントしたままにする。 */}
                {promptsMounted && <PromptSettingsPanel />}
              </SettingsPanel>
              <SettingsPanel active={activeTab === "display"} tab="display">
                <DisplaySettingsPanel />
              </SettingsPanel>
              <SettingsPanel active={activeTab === "debug"} tab="debug">
                {/* 他のパネルと違い、これはアクティブな間だけマウントする。レポート
                    の収集はデータベースと設定ファイルを読むので、ダイアログを開いた
                    だけでそれを行ってはならない。 */}
                {activeTab === "debug" && <DebugSettingsPanel />}
              </SettingsPanel>
              <SettingsPanel active={activeTab === "licenses"} tab="licenses">
                <LicensesSettingsPanel />
              </SettingsPanel>
            </>,
          )}
          <footer>
            {/* 保存の報告は、そのとき書き込んだ内容についてのものである。次の
                編集を始めたら、その報告はもう当てはまらない。 */}
            <SaveStatus pending={saving} saved={saved && !sections.dirty} />
            <SaveButton
              dirty={sections.dirty && !sections.invalid}
              pending={saving}
              onClick={() => {
                void save();
              }}
            />
          </footer>
        </section>
      </div>
      {discarding && (
        <DiscardChangesDialog
          onCancel={() => {
            setDiscarding(false);
          }}
          onConfirm={onClose}
        />
      )}
    </>
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
  const language = (i18n.resolvedLanguage ?? "en") as SupportedLanguage;
  const [draft, setDraft] = useState<{
    language: SupportedLanguage;
    theme: ThemePreference;
  }>();
  const saved = { language, theme: readThemePreference() };
  const current = draft ?? saved;

  useRegisterSettingsSection("display", {
    dirty: current.language !== saved.language || current.theme !== saved.theme,
    invalid: false,
    save: async () => {
      setDisplayTheme(current.theme);
      await setDisplayLanguage(current.language);
      setDraft(undefined);
    },
  });

  return (
    <div className="settings-display-list">
      <label className="settings-display-row">
        <span className="settings-display-copy">
          <strong>{t("settings.display.language.label")}</strong>
        </span>
        <select
          aria-label={t("settings.display.language.label")}
          value={current.language}
          onChange={(event) => {
            setDraft({
              ...current,
              language: event.target.value as SupportedLanguage,
            });
          }}
        >
          {supportedLanguages.map((option) => (
            <option value={option} key={option}>
              {t(`settings.display.language.options.${option}`)}
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
          value={current.theme}
          onChange={(event) => {
            setDraft({
              ...current,
              theme: event.target.value as ThemePreference,
            });
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
  );
}
