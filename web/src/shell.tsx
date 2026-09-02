import { Link, useLocation, useNavigate } from "@tanstack/react-router";
import { Plus, Settings, X } from "lucide-react";
import {
  useEffect,
  useState,
  type ReactNode,
  type SyntheticEvent,
} from "react";
import { useTranslation } from "react-i18next";
import { mutations } from "./api";
import { isDemoMode } from "./demo";
import {
  featureCategories,
  featureCategoryById,
  featureCategoryForPath,
} from "./feature-status";
import { formValue } from "./form";
import { useAutoSync, useDomainMutation, useSnapshot } from "./hooks";
import { formatError } from "./i18n/domain";
import { readFeatureCategory, writeFeatureCategory } from "./i18n/settings";
import { AutoSyncStatusContext } from "./sync-status";
import { appVersion } from "./version";
import { IconButton } from "./views/IconButton";
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
  const pathname = useLocation({ select: (location) => location.pathname });
  const features = snapshot.data?.features;
  const selected = useSelectedFeatureCategory(pathname);
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
          <span className="brand-line">
            <span className="brand-mark">
              P<span>R</span>X
            </span>
            <span className="app-version">v{appVersion()}</span>
          </span>
        </Link>
        <nav aria-label={t("nav.features")}>
          <Link to="/" className="nav-link">
            {t("nav.overview")}
          </Link>
          <hr className="nav-divider" />
          {featureCategories.map((category) => (
            <Link
              key={category.id}
              to={category.path}
              className="nav-link"
              // The selection is the sidebar's own, not the matched route, so
              // it is marked here rather than through activeProps.
              data-active={category.id === selected.id || undefined}
              aria-current={category.id === selected.id ? "page" : undefined}
            >
              {t(category.navLabelKey)}{" "}
              <span>{features?.filter(category.select).length ?? "—"}</span>
            </Link>
          ))}
          <hr className="nav-divider" />
          <Link to="/tasks" search={{ q: "" }} className="nav-link">
            {t("nav.taskSearch")}
          </Link>
          {features?.filter(selected.select).map((feature) => (
            <Link
              key={feature.id}
              to="/features/$featureId"
              params={{ featureId: feature.id }}
              className="feature-link"
              activeProps={{ "data-active": true }}
            >
              <i
                className={
                  feature.conflictCount
                    ? "pulse conflict"
                    : feature.readyCount
                      ? "pulse ready"
                      : "pulse"
                }
              />
              <span>{feature.title}</span>
              <b>
                {feature.mergedCount}/{feature.taskCount}
              </b>
            </Link>
          ))}
        </nav>
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

// The sidebar keeps its own selection so that the overview, task search, and
// the feature workspaces do not move it. Opening a category's own list page is
// the one navigation that adopts a category, whether it came from the sidebar,
// a shared link, or browser history, and that route is also what records it.
// Navigating always re-renders the shell, so the stored value is in place
// before the next route reads it and no copy has to live in component state.
function useSelectedFeatureCategory(pathname: string) {
  const routeCategory = featureCategoryForPath(pathname);
  const routeCategoryId = routeCategory?.id;
  useEffect(() => {
    if (routeCategoryId) writeFeatureCategory(routeCategoryId);
  }, [routeCategoryId]);
  return routeCategory ?? featureCategoryById(readFeatureCategory());
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

function FeatureCreateDialog({ onClose }: { onClose: () => void }) {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const createFeature = useDomainMutation(mutations.createFeature);

  async function submit(event: SyntheticEvent<HTMLFormElement>) {
    event.preventDefault();
    const data = new FormData(event.currentTarget);
    const response = await createFeature.mutateAsync({
      slug: formValue(data, "slug"),
      title: formValue(data, "title"),
      description: formValue(data, "description"),
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
          {t("common.slug")}
          <input
            name="slug"
            required
            pattern="[a-z0-9]+(?:-[a-z0-9]+)*"
            placeholder={t("featureCreate.slugPlaceholder")}
          />
        </label>
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
