import { useTranslation } from "react-i18next";

export function CreditsSettingsPanel() {
  const { t } = useTranslation();
  return (
    <div className="settings-credit-list">
      <p className="settings-credit-row">
        <a
          href="https://reactflow.dev/attribution"
          target="_blank"
          rel="noreferrer"
        >
          {t("settings.credits.reactFlowName")}
        </a>
      </p>
    </div>
  );
}
