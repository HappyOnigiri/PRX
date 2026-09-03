import { useNavigate, useParams } from "@tanstack/react-router";
import { ArrowLeft, Pencil } from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import type { Document, Feature, Project } from "../gen/prx/v1/prx_pb";
import { useSnapshot } from "../hooks";
import { documentsInProject, featuresInProject } from "../project";
import { CopyableIdentifier } from "./CopyableIdentifier";
import { DocumentReferences } from "./DocumentReferences";
import { EditProjectDialog } from "./EditProjectDialog";
import { IconButton } from "./IconButton";
import { MarkdownPreview } from "./MarkdownPreview";
import { ProjectFeatureList } from "./ProjectFeatureList";
import { type TaskNodeDocument } from "./TaskNode";

export function ProjectWorkspace() {
  const { t } = useTranslation();
  const { projectId } = useParams({ from: "/projects/$projectId" });
  const navigate = useNavigate();
  const snapshot = useSnapshot();
  const [showEdit, setShowEdit] = useState(false);
  const [previewDocument, setPreviewDocument] = useState<TaskNodeDocument>();
  const data = snapshot.data;
  const project = data?.projects.find((item) => item.id === projectId);

  if (snapshot.isPending)
    return (
      <div className="state-message">
        <div className="spinner" />
        <h1>{t("project.workspaceLoading")}</h1>
      </div>
    );
  if (!project || !data)
    return (
      <div className="state-message">
        <h1>{t("project.notFound")}</h1>
        <IconButton
          icon={ArrowLeft}
          label={t("project.returnList")}
          variant="secondary"
          onClick={() =>
            void navigate({ to: "/projects", search: { archived: false } })
          }
        />
      </div>
    );

  return (
    <ProjectContent
      project={project}
      features={featuresInProject(data.features, projectId)}
      documents={documentsInProject(data.documents, projectId)}
      previewDocument={previewDocument}
      showEdit={showEdit}
      onPreviewDocument={setPreviewDocument}
      onClosePreview={() => {
        setPreviewDocument(undefined);
      }}
      onEdit={() => {
        setShowEdit(true);
      }}
      onCloseEdit={() => {
        setShowEdit(false);
      }}
      onDeleted={() => {
        void navigate({ to: "/projects", search: { archived: false } });
      }}
    />
  );
}

interface ProjectContentProps {
  project: Project;
  features: Feature[];
  documents: Document[];
  previewDocument: TaskNodeDocument | undefined;
  showEdit: boolean;
  onPreviewDocument: (document: TaskNodeDocument) => void;
  onClosePreview: () => void;
  onEdit: () => void;
  onCloseEdit: () => void;
  onDeleted: () => void;
}

function ProjectContent(props: ProjectContentProps) {
  const { t } = useTranslation();
  const archived = props.project.archived;
  return (
    <div className={archived ? "workspace is-archived" : "workspace"}>
      <header className="workspace-head">
        <div
          className="workspace-title"
          title={props.project.description || t("project.noDescription")}
        >
          <h1>{props.project.title}</h1>
          <CopyableIdentifier
            label={t("project.membership")}
            value={props.project.id}
            valueOnly
          />
          <p className="eyebrow">
            {t("project.slugEyebrow", { slug: props.project.slug })}
          </p>
        </div>
        <div className="workspace-actions">
          <DocumentReferences
            parent={{ projectId: props.project.id }}
            documents={props.documents}
            onPreview={props.onPreviewDocument}
            readOnly={archived}
          />
          <IconButton
            icon={Pencil}
            label={
              archived ? t("project.manageProject") : t("project.editProject")
            }
            variant="secondary"
            iconOnly
            onClick={props.onEdit}
          />
        </div>
      </header>
      {archived && (
        <div className="archived-notice" role="status">
          <strong>{t("project.archivedLabel")}</strong>
          <span>{t("project.archivedDetail")}</span>
        </div>
      )}
      <div className="workspace-body project-body">
        <ProjectFeatureList features={props.features} />
      </div>
      <ProjectOverlays props={props} />
    </div>
  );
}

function ProjectOverlays({ props }: { props: ProjectContentProps }) {
  return (
    <>
      {props.previewDocument && (
        <MarkdownPreview
          key={props.previewDocument.id}
          document={props.previewDocument}
          onClose={props.onClosePreview}
        />
      )}
      {props.showEdit && (
        <EditProjectDialog
          project={props.project}
          referenceCount={props.documents.length}
          onClose={props.onCloseEdit}
          onDeleted={props.onDeleted}
        />
      )}
    </>
  );
}
