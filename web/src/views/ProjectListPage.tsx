import { Link, useNavigate, useSearch } from "@tanstack/react-router";
import { Archive, ArchiveRestore, Plus } from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import type { Project } from "../gen/prx/v1/prx_pb";
import { useSnapshot } from "../hooks";
import { formatError } from "../i18n/domain";
import { featuresInProject, projectsByArchive } from "../project";
import { StateMessage } from "./Dashboard";
import { IconButton } from "./IconButton";
import { ProjectCreateDialog } from "./ProjectCreateDialog";

export function ProjectListPage() {
  const { t } = useTranslation();
  const { archived } = useSearch({ from: "/projects" });
  const navigate = useNavigate();
  const [showCreate, setShowCreate] = useState(false);
  const { data, isPending, error, refetch } = useSnapshot();

  if (isPending)
    return (
      <StateMessage
        title={t("project.loadingTitle")}
        detail={t("project.loadingDetail")}
      />
    );
  if (error)
    return (
      <StateMessage
        title={t("project.errorTitle")}
        detail={formatError(error, t)}
        action={() => void refetch()}
      />
    );

  const projects = projectsByArchive(data.projects, archived);
  return (
    <div className="dashboard project-list-page">
      <header className="page-head">
        <div>
          <p className="section-label">{t("project.eyebrow")}</p>
          <h1>{t("project.title")}</h1>
          <p>{t("project.description")}</p>
        </div>
        <div className="page-head-status project-list-actions">
          <IconButton
            icon={archived ? ArchiveRestore : Archive}
            label={
              archived ? t("project.showActive") : t("project.showArchived")
            }
            variant="secondary"
            onClick={() => {
              // The toggle lives in the URL so reload, history, and a shared
              // link reproduce the view, as task search does with its query.
              void navigate({
                to: "/projects",
                search: { archived: !archived },
              });
            }}
          />
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
      <section className="feature-list" aria-label={t("project.listLabel")}>
        {projects.length === 0 ? (
          <div className="empty compact">
            <h2>
              {t(
                archived ? "project.emptyArchivedTitle" : "project.emptyTitle",
              )}
            </h2>
            <p>
              {t(
                archived
                  ? "project.emptyArchivedDetail"
                  : "project.emptyDetail",
              )}
            </p>
          </div>
        ) : (
          projects.map((project) => (
            <ProjectListRow
              key={project.id}
              project={project}
              featureCount={featuresInProject(data.features, project.id).length}
            />
          ))
        )}
      </section>
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
      className="feature-list-row project-list-row"
    >
      <div className="feature-list-row-title">
        <b>{project.title}</b>
        <small>{project.slug}</small>
      </div>
      <span>{t("project.featureCount", { count: featureCount })}</span>
      <strong>
        {project.archived ? t("project.archivedBadge") : project.id}
      </strong>
    </Link>
  );
}
