import { Link } from "@tanstack/react-router";
import { Settings } from "lucide-react";
import { useState, type ReactNode } from "react";
import { useTranslation } from "react-i18next";
import { isDemoMode } from "./demo";
import type { Feature, Project } from "./gen/prx/v1/prx_pb";
import { useAutoSync, useSnapshot } from "./hooks";
import { projectsByArchive } from "./project";
import { AutoSyncStatusContext } from "./sync-status";
import { IconButton } from "./views/IconButton";
import { ProjectTree } from "./views/ProjectTree";
import { SettingsDialog } from "./views/SettingsDialog";

export function AppShell({ children }: { children: ReactNode }) {
  const autoSync = useAutoSync(true);
  return (
    <AutoSyncStatusContext.Provider value={autoSync}>
      <AppShellLayout>{children}</AppShellLayout>
    </AutoSyncStatusContext.Provider>
  );
}

function AppShellLayout({ children }: { children: ReactNode }) {
  const { t } = useTranslation();
  const snapshot = useSnapshot();
  const [showSettings, setShowSettings] = useState(false);
  const features = snapshot.data?.features;
  const projects = snapshot.data?.projects;
  const demo = isDemoMode();
  return (
    <div className="app-shell" data-demo={demo || undefined}>
      {demo && (
        <div className="demo-banner" role="status">
          <span className="demo-banner-full">
            DEMO — Changes reset on restart / 変更は再起動時にリセットされます
          </span>
          <span className="demo-banner-compact">
            <span>DEMO · Reset on restart</span>
            <span>再起動でリセット</span>
          </span>
        </div>
      )}
      <aside className="rail">
        <Link to="/" className="brand" aria-label={t("nav.dashboard")}>
          <span className="brand-mark">
            P<span>R</span>X
          </span>
        </Link>
        <RailNavigation features={features} projects={projects} />
        <RailSettings
          onOpenSettings={() => {
            setShowSettings(true);
          }}
        />
        {snapshot.isError && (
          <div className="rail-foot">
            <span className="rail-health">
              <span className="health bad" />
              {t("nav.serverUnavailable")}
            </span>
          </div>
        )}
      </aside>
      <main className="main-stage">{children}</main>
      {showSettings && (
        <SettingsDialog
          onClose={() => {
            setShowSettings(false);
          }}
        />
      )}
    </div>
  );
}

// ツリーは画面リンクと同じ <nav> の中に置く。.rail の直下に要素を足すと、900px
// と 600px のレイアウトが定義するグリッドが崩れるため。
function RailNavigation({
  features,
  projects,
}: {
  features: Feature[] | undefined;
  projects: Project[] | undefined;
}) {
  const { t } = useTranslation();
  const activeProjects = projects ? projectsByArchive(projects, false) : [];
  return (
    <nav aria-label={t("nav.primary")}>
      <Link to="/" className="nav-link">
        {t("nav.overview")}
      </Link>
      <Link to="/tasks" search={{ q: "" }} className="nav-link">
        {t("nav.taskSearch")}
      </Link>
      <hr className="nav-divider" />
      <Link
        to="/projects"
        search={{ archived: false }}
        className="nav-link"
        id="nav-projects-heading"
        // 既定の前方一致だと個々の project ページでも見出しが点灯し、search まで
        // 一致条件に含めると同じ画面であるアーカイブ表示で消灯してしまう。
        activeOptions={{ exact: true, includeSearch: false }}
        activeProps={{ "data-active": true }}
      >
        {t("nav.projects")}{" "}
        <span>{projects ? activeProjects.length : "—"}</span>
      </Link>
      {features && (
        <ProjectTree
          headingId="nav-projects-heading"
          projects={activeProjects}
          features={features}
        />
      )}
      <hr className="nav-divider" />
    </nav>
  );
}

function RailSettings({ onOpenSettings }: { onOpenSettings: () => void }) {
  const { t } = useTranslation();
  return (
    <IconButton
      icon={Settings}
      label={t("settings.open")}
      variant="secondary"
      className="settings-trigger"
      onClick={onOpenSettings}
    />
  );
}
