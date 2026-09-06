import { Link, useNavigate, useSearch } from "@tanstack/react-router";
import { Plus } from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import type { Feature, Project } from "../gen/prx/v1/prx_pb";
import { featuresInProject, projectsByArchive } from "../project";
import { EntityIcon } from "./EntityIcon";
import { IconButton } from "./IconButton";
import { ProjectCreateDialog } from "./ProjectCreateDialog";
import { TabList, TabPanel } from "./TabList";
import { useProjectSnapshot } from "./useProjectSnapshot";

export function ProjectListPage() {
  const { t } = useTranslation();
  const { archived } = useSearch({ from: "/projects" });
  const navigate = useNavigate();
  const [showCreate, setShowCreate] = useState(false);
  const snapshot = useProjectSnapshot();
  if (snapshot.message) return snapshot.message;
  const data = snapshot.data;

  return (
    <div className="dashboard project-list-page">
      <header className="page-head">
        <div>
          <h1>{t("project.title")}</h1>
        </div>
        <div className="page-head-status project-list-actions">
          <IconButton
            icon={Plus}
            label={t("nav.newProject")}
            variant="primary"
            onClick={() => {
              setShowCreate(true);
            }}
          />
        </div>
      </header>
      {/* A project is either in play or archived; it has no completed state,
          so this strip carries two tabs where a feature list carries three. */}
      <TabList
        tabs={[
          { id: "active", label: t("project.tabs.active") },
          { id: "archived", label: t("project.tabs.archived") },
        ]}
        active={archived ? "archived" : "active"}
        onSelect={(id) => {
          // The selection lives in the URL so reload, history, and a shared
          // link reproduce the view, as task search does with its query.
          void navigate({
            to: "/projects",
            search: { archived: id === "archived" },
          });
        }}
        idPrefix="project-list"
        className="workspace-tabs"
        tabClassName="workspace-tab"
        label={t("project.tabsLabel")}
      />
      {(["active", "archived"] as const).map((id) => (
        <TabPanel
          key={id}
          active={id === (archived ? "archived" : "active")}
          className="workspace-tab-panel"
          idPrefix="project-list"
          tab={id}
        >
          <section className="feature-list" aria-label={t("project.listLabel")}>
            <ProjectListPanel
              projects={projectsByArchive(data.projects, id === "archived")}
              features={data.features}
              archived={id === "archived"}
            />
          </section>
        </TabPanel>
      ))}
      {showCreate && (
        <ProjectCreateDialog
          onClose={() => {
            setShowCreate(false);
          }}
        />
      )}
    </div>
  );
}

function ProjectListPanel({
  projects,
  features,
  archived,
}: {
  projects: Project[];
  features: Feature[];
  archived: boolean;
}) {
  const { t } = useTranslation();
  if (projects.length === 0)
    return (
      <div className="empty compact">
        <h2>
          {t(archived ? "project.emptyArchivedTitle" : "project.emptyTitle")}
        </h2>
        <p>
          {t(archived ? "project.emptyArchivedDetail" : "project.emptyDetail")}
        </p>
      </div>
    );
  return projects.map((project) => (
    <ProjectListRow
      key={project.id}
      project={project}
      featureCount={featuresInProject(features, project.id).length}
    />
  ));
}

function ProjectListRow({
  project,
  featureCount,
}: {
  project: Project;
  featureCount: number;
}) {
  const { t } = useTranslation();
  return (
    <Link
      to="/projects/$projectId"
      params={{ projectId: project.id }}
      search={{ features: project.archived ? "archived" : "active" }}
      className="feature-list-row project-list-row"
    >
      <div className="feature-list-row-title">
        <b>
          <EntityIcon kind="project" size={15} />
          {project.title}
        </b>
      </div>
      <span>{t("project.featureCount", { count: featureCount })}</span>
      {/* The slot a feature row fills with a status carries only the archived
          badge here, because the active tab already says every row is active
          and the identifier belongs to the workspace, where it can be copied. */}
      {project.archived && <strong>{t("project.archivedBadge")}</strong>}
    </Link>
  );
}
