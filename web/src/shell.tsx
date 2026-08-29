import { type ReactNode, useState } from "react";
import { Link } from "@tanstack/react-router";
import { useTranslation } from "react-i18next";
import { useSnapshot } from "./hooks";
import { FeatureCreateDialog } from "./shell/FeatureCreateDialog";
import { FeatureNavigation } from "./shell/FeatureNavigation";
import { LanguageSelector } from "./shell/LanguageSelector";
import { ThemeSelector } from "./shell/ThemeSelector";

export function AppShell({ children }: { children: ReactNode }) {
  const { t } = useTranslation();
  const snapshot = useSnapshot();
  const [showCreate, setShowCreate] = useState(false);

  return (
    <div className="app-shell">
      <aside className="rail">
        <Link to="/" className="brand" aria-label={t("nav.dashboard")}>
          <span className="brand-mark">
            P<span>R</span>X
          </span>
          <small>{t("nav.dependencyControl")}</small>
        </Link>
        <FeatureNavigation features={snapshot.data?.features} />
        <button
          className="rail-action"
          onClick={() => {
            setShowCreate(true);
          }}
        >
          {t("nav.newFeature")}
        </button>
        <div className="rail-settings">
          <LanguageSelector />
          <ThemeSelector />
        </div>
        <div className="rail-foot">
          <span className={snapshot.isError ? "health bad" : "health"} />
          {snapshot.isError
            ? t("nav.serverUnavailable")
            : t("nav.localDatabaseOnline")}
        </div>
      </aside>
      <main className="main-stage">{children}</main>
      {showCreate && (
        <FeatureCreateDialog
          onClose={() => {
            setShowCreate(false);
          }}
        />
      )}
    </div>
  );
}
