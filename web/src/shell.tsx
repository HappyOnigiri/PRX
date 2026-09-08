import { Link } from "@tanstack/react-router";
import { PanelLeftClose, PanelLeftOpen, Settings, X } from "lucide-react";
import {
  useEffect,
  useRef,
  useState,
  type CSSProperties,
  type ReactNode,
} from "react";
import { useTranslation } from "react-i18next";
import {
  isDemoMode,
  readDemoNoticeDismissed,
  writeDemoNoticeDismissed,
} from "./demo";
import type { Feature, Project } from "./gen/prx/v1/prx_pb";
import { useAutoSync, useSnapshot } from "./hooks";
import {
  readRailCollapsed,
  readRailWidth,
  writeRailCollapsed,
} from "./i18n/settings";
import { projectsByArchive } from "./project";
import { AutoSyncStatusContext } from "./sync-status";
import { IconButton } from "./views/IconButton";
import { ProjectTree } from "./views/ProjectTree";
import { RailResizer } from "./views/RailResizer";
import { SettingsDialog } from "./views/SettingsDialog";

const railId = "prx-rail";

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
  const [railWidth, setRailWidth] = useState(readRailWidth);
  const [railCollapsed, setRailCollapsed] = useState(readRailCollapsed);
  const [railResizing, setRailResizing] = useState(false);
  const { collapseRef, restoreRef, markUserToggle } =
    useRailToggleFocus(railCollapsed);
  const features = snapshot.data?.features;
  const projects = snapshot.data?.projects;
  // 消した警告は demo のサーバプロセス単位で覚える。読み込み直しでは戻らず、
  // demo を起動し直すと戻る。
  const [demoDismissed, setDemoDismissed] = useState(readDemoNoticeDismissed);
  const demo = isDemoMode() && !demoDismissed;

  function toggleRail(collapsed: boolean) {
    markUserToggle();
    setRailCollapsed(collapsed);
    writeRailCollapsed(collapsed);
  }

  return (
    <div
      className="app-shell"
      data-demo={demo || undefined}
      data-rail-collapsed={railCollapsed || undefined}
      data-rail-resizing={railResizing || undefined}
      style={{ "--rail-width": `${railWidth}px` } as CSSProperties}
    >
      {demo && (
        <div className="demo-banner" role="status">
          <span className="demo-banner-full">
            DEMO — Changes reset on restart / 変更は再起動時にリセットされます
          </span>
          <span className="demo-banner-compact">
            <span>DEMO · Reset on restart</span>
            <span>再起動でリセット</span>
          </span>
          <IconButton
            className="demo-banner-dismiss"
            icon={X}
            iconOnly
            label={t("demo.dismiss")}
            size="compact"
            variant="quiet"
            onClick={() => {
              setDemoDismissed(true);
              writeDemoNoticeDismissed();
            }}
          />
        </div>
      )}
      <aside className="rail" id={railId}>
        <div className="rail-head">
          <Link to="/" className="brand" aria-label={t("nav.dashboard")}>
            <span className="brand-mark">
              P<span>R</span>X
            </span>
          </Link>
          <IconButton
            aria-controls={railId}
            aria-expanded
            className="rail-collapse"
            icon={PanelLeftClose}
            iconOnly
            label={t("nav.hideRail")}
            ref={collapseRef}
            variant="quiet"
            onClick={() => {
              toggleRail(true);
            }}
          />
        </div>
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
        <RailResizer
          railId={railId}
          width={railWidth}
          onResizing={setRailResizing}
          onWidth={setRailWidth}
        />
      </aside>
      {/* 復元ボタンは main の外に置く。main ランドマークの中身をページ内容だけ
          に保て、DOM 順が先なので Tab で最初に当たる。 */}
      <IconButton
        aria-controls={railId}
        aria-expanded={false}
        className="rail-restore"
        icon={PanelLeftOpen}
        iconOnly
        label={t("nav.showRail")}
        ref={restoreRef}
        variant="secondary"
        onClick={() => {
          toggleRail(false);
        }}
      />
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

// 折りたたむと押したボタンが display: none になり、フォーカスが body へ落ちる。
// 相手側のボタンへ移すが、保存済みの最小化状態で開いた初回描画では奪わない。
function useRailToggleFocus(collapsed: boolean) {
  const collapseRef = useRef<HTMLButtonElement>(null);
  const restoreRef = useRef<HTMLButtonElement>(null);
  const userToggled = useRef(false);

  useEffect(() => {
    if (!userToggled.current) return;
    userToggled.current = false;
    const target = collapsed ? restoreRef.current : collapseRef.current;
    target?.focus();
  }, [collapsed]);

  return {
    collapseRef,
    restoreRef,
    markUserToggle: () => {
      userToggled.current = true;
    },
  };
}
