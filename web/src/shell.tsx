import { Link, useNavigate } from "@tanstack/react-router";
import { FormEvent, useState, type ReactNode } from "react";
import { useTranslation } from "react-i18next";
import { mutations } from "./api";
import { formValue } from "./form";
import { useDomainMutation, useSnapshot } from "./hooks";
import { setDisplayLanguage } from "./i18n";
import { formatError } from "./i18n/domain";
import {
  readThemePreference,
  supportedLanguages,
  themePreferences,
  type SupportedLanguage,
  type ThemePreference,
} from "./i18n/settings";
import { setDisplayTheme } from "./theme";

export function AppShell({ children }: { children: ReactNode }) {
  const { t, i18n } = useTranslation();
  const snapshot = useSnapshot();
  const navigate = useNavigate();
  const [showCreate, setShowCreate] = useState(false);
  const [theme, setTheme] = useState(readThemePreference);
  const createFeature = useDomainMutation(mutations.createFeature);
  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const data = new FormData(event.currentTarget);
    const response = await createFeature.mutateAsync({
      slug: formValue(data, "slug"),
      title: formValue(data, "title"),
      description: formValue(data, "description"),
    });
    setShowCreate(false);
    if (response.feature)
      await navigate({
        to: "/features/$featureId",
        params: { featureId: response.feature.id },
      });
  }
  return (
    <div className="app-shell">
      <aside className="rail">
        <Link to="/" className="brand" aria-label={t("nav.dashboard")}>
          <span className="brand-mark">
            P<span>R</span>X
          </span>
          <small>{t("nav.dependencyControl")}</small>
        </Link>
        <nav aria-label={t("nav.features")}>
          <Link to="/" className="nav-link">
            {t("nav.overview")}{" "}
            <span>{snapshot.data?.features.length ?? "—"}</span>
          </Link>
          <div className="nav-caption">{t("nav.activeCircuits")}</div>
          {snapshot.data?.features
            .filter((f) => !f.archived)
            .map((feature) => (
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
        <button className="rail-action" onClick={() => setShowCreate(true)}>
          {t("nav.newFeature")}
        </button>
        <div className="rail-settings">
          <label className="language-setting">
            <span>{t("language.label")}</span>
            <select
              aria-label={t("language.label")}
              value={i18n.resolvedLanguage ?? "en"}
              onChange={(event) =>
                void setDisplayLanguage(event.target.value as SupportedLanguage)
              }
            >
              {supportedLanguages.map((language) => (
                <option value={language} key={language}>
                  {t(`language.${language}`)}
                </option>
              ))}
            </select>
          </label>
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
              <p className="form-error">
                {formatError(createFeature.error, t)}
              </p>
            )}
            <footer>
              <button
                type="button"
                className="secondary"
                onClick={() => setShowCreate(false)}
              >
                {t("common.cancel")}
              </button>
              <button disabled={createFeature.isPending}>
                {t("featureCreate.submit")}
              </button>
            </footer>
          </form>
        </div>
      )}
    </div>
  );
}
