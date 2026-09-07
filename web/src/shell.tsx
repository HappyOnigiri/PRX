import { Link, useNavigate } from "@tanstack/react-router";
import { Plus, Settings, X } from "lucide-react";
import { useState, type ReactNode, type SyntheticEvent } from "react";
import { useTranslation } from "react-i18next";
import { mutations } from "./api";
import { isDemoMode } from "./demo";
import { formValue } from "./form";
import type { Feature, Project } from "./gen/prx/v1/prx_pb";
import { useAutoSync, useDomainMutation, useSnapshot } from "./hooks";
import { formatError } from "./i18n/domain";
import { projectsByArchive } from "./project";
import { AutoSyncStatusContext } from "./sync-status";
import { IconButton } from "./views/IconButton";
import { ProjectSelectField } from "./views/ProjectSelectField";
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
  const [showCreate, setShowCreate] = useState(false);
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
        <IconButton
          icon={Plus}
          label={t("nav.newFeature")}
          variant="primary"
          className="rail-action"
          onClick={() => {
            setShowCreate(true);
          }}
        />
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
      {showCreate && (
        <FeatureCreateDialog
          projects={projects ?? []}
          onClose={() => {
            setShowCreate(false);
          }}
        />
      )}
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

// The tree sits inside the same <nav> as the screen links: adding an element
// straight under .rail would break the grid the 900px and 600px layouts
// define.
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
        // The default prefix match would light the heading up on a project's
        // own page, and matching the search would put it out on the archived
        // view, which is the same screen.
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

function FeatureCreateDialog({
  projects,
  onClose,
}: {
  projects: Project[];
  onClose: () => void;
}) {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const createFeature = useDomainMutation(mutations.createFeature);

  async function submit(event: SyntheticEvent<HTMLFormElement>) {
    event.preventDefault();
    const data = new FormData(event.currentTarget);
    const response = await createFeature.mutateAsync({
      title: formValue(data, "title"),
      description: formValue(data, "description"),
      projectId: formValue(data, "projectId"),
    });
    onClose();
    if (response.feature)
      await navigate({
        to: "/features/$featureId",
        params: { featureId: response.feature.id },
      });
  }

  return (
    <div className="scrim" role="presentation">
      <form
        className="dialog"
        onSubmit={submit}
        aria-label={t("featureCreate.formLabel")}
      >
        <header>
          <h2>{t("featureCreate.title")}</h2>
        </header>
        <label>
          {t("common.title")}
          <input
            name="title"
            required
            placeholder={t("featureCreate.titlePlaceholder")}
          />
        </label>
        <label>
          {t("common.description")}
          <textarea
            name="description"
            placeholder={t("featureCreate.descriptionPlaceholder")}
          />
        </label>
        <ProjectSelectField projects={projects} />
        {createFeature.error && (
          <p className="form-error">{formatError(createFeature.error, t)}</p>
        )}
        <footer>
          <IconButton
            icon={X}
            label={t("common.cancel")}
            variant="secondary"
            onClick={onClose}
          />
          <IconButton
            icon={Plus}
            label={t("featureCreate.submit")}
            variant="primary"
            type="submit"
            disabled={createFeature.isPending}
          />
        </footer>
      </form>
    </div>
  );
}
