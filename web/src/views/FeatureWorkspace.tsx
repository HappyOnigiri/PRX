import { Link, useNavigate, useParams } from "@tanstack/react-router";
import { ArrowLeft, Pencil, Plus, RefreshCw } from "lucide-react";
import { useCallback, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { mutations } from "../api";
import type {
  Dependency,
  Feature,
  Project,
  PullRequest,
  Snapshot,
  Task,
} from "../gen/prx/v1/prx_pb";
import { useDomainMutation, useSnapshot } from "../hooks";
import { AddDocumentDialog } from "./AddDocumentDialog";
import { CopyableIdentifier } from "./CopyableIdentifier";
import { CreateTaskDialog } from "./CreateTaskDialog";
import { DocumentReferences } from "./DocumentReferences";
import { EditFeatureDialog } from "./EditFeatureDialog";
import { FeatureGraph } from "./FeatureGraph";
import { IconButton } from "./IconButton";
import { MarkdownPreview } from "./MarkdownPreview";
import { TaskInspector } from "./TaskInspector";
import { type TaskNodeDocument } from "./TaskNode";

interface DocumentTarget {
  taskId: string;
  trigger: HTMLButtonElement;
}

export function FeatureWorkspace() {
  const { t } = useTranslation();
  const { featureId } = useParams({ from: "/features/$featureId" });
  const navigate = useNavigate();
  const snapshot = useSnapshot();
  const [selected, setSelected] = useState<string>();
  const [showTask, setShowTask] = useState(false);
  const [showFeatureEdit, setShowFeatureEdit] = useState(false);
  const [previewDocument, setPreviewDocument] = useState<TaskNodeDocument>();
  const [documentTarget, setDocumentTarget] = useState<DocumentTarget>();
  const data = snapshot.data;
  const feature = data?.features.find((item) => item.id === featureId);
  const project = data?.projects.find((item) => item.id === feature?.projectId);
  const {
    tasks,
    dependencies,
    pullRequests,
    documentsByTask,
    featureDocuments,
  } = useFeatureWorkspaceData(data, featureId);
  const openTaskDialog = useCallback(() => {
    setShowTask(true);
  }, []);
  const openDocumentDialog = useCallback(
    (taskId: string, trigger: HTMLButtonElement) => {
      setDocumentTarget({ taskId, trigger });
    },
    [],
  );
  const sync = useDomainMutation((id: string) => mutations.sync(id));

  if (snapshot.isPending)
    return (
      <div className="state-message">
        <div className="spinner" />
        <h1>{t("workspace.loading")}</h1>
      </div>
    );
  if (!feature || !data)
    return (
      <div className="state-message">
        <h1>{t("workspace.notFound")}</h1>
        <IconButton
          icon={ArrowLeft}
          label={t("workspace.returnOverview")}
          variant="secondary"
          onClick={() => void navigate({ to: "/" })}
        />
      </div>
    );

  const selectedTask = tasks.find((task) => task.id === selected);
  return (
    <WorkspaceContent
      feature={feature}
      featureId={featureId}
      project={project}
      projects={data.projects}
      tasks={tasks}
      dependencies={dependencies}
      pullRequests={pullRequests}
      documentsByTask={documentsByTask}
      featureDocuments={featureDocuments}
      selectedTask={selectedTask}
      previewDocument={previewDocument}
      documentTarget={documentTarget}
      showTask={showTask}
      showFeatureEdit={showFeatureEdit}
      syncPending={sync.isPending}
      onSync={() => {
        sync.mutate(featureId);
      }}
      onCreateTask={openTaskDialog}
      onEditTask={setSelected}
      onPreviewDocument={setPreviewDocument}
      onAddDocument={openDocumentDialog}
      onEditFeature={() => {
        setShowFeatureEdit(true);
      }}
      onCloseInspector={() => {
        setSelected(undefined);
      }}
      onClosePreview={() => {
        setPreviewDocument(undefined);
      }}
      onCloseDocumentDialog={() => {
        setDocumentTarget(undefined);
      }}
      onCloseTask={() => {
        setShowTask(false);
      }}
      onCloseFeatureEdit={() => {
        setShowFeatureEdit(false);
      }}
      onFeatureDeleted={() => {
        void navigate({ to: feature.readOnly ? "/archived" : "/" });
      }}
    />
  );
}

// The workspace narrows one snapshot to a single feature. Keeping the memos
// together lets the component itself stay about rendering and state.
function useFeatureWorkspaceData(
  data: Snapshot | undefined,
  featureId: string,
) {
  const tasks = useMemo(
    () => data?.tasks.filter((task) => task.featureId === featureId) ?? [],
    [data, featureId],
  );
  const taskIds = useMemo(() => new Set(tasks.map((task) => task.id)), [tasks]);
  const dependencies = useMemo(
    () =>
      data?.dependencies.filter((dependency) =>
        taskIds.has(dependency.blockerTaskId),
      ) ?? [],
    [data, taskIds],
  );
  const pullRequests = useMemo(
    () => new Map(data?.pullRequests.map((pr) => [pr.taskId, pr]) ?? []),
    [data],
  );
  const documentsByTask = useMemo(() => {
    const result = new Map<string, TaskNodeDocument[]>();
    for (const document of data?.documents ?? []) {
      if (!document.taskId) continue;
      const documents = result.get(document.taskId) ?? [];
      documents.push(document);
      result.set(document.taskId, documents);
    }
    return result;
  }, [data]);
  // A document with no feature carries an empty ID rather than an absent one,
  // so the comparison has to reject the empty value.
  const featureDocuments = useMemo(
    () =>
      data?.documents.filter(
        (document) =>
          document.featureId !== "" && document.featureId === featureId,
      ) ?? [],
    [data, featureId],
  );
  return {
    tasks,
    dependencies,
    pullRequests,
    documentsByTask,
    featureDocuments,
  };
}

interface WorkspaceContentProps {
  feature: Feature;
  featureId: string;
  project: Project | undefined;
  projects: Project[];
  tasks: Task[];
  dependencies: Dependency[];
  pullRequests: Map<string, PullRequest>;
  documentsByTask: Map<string, TaskNodeDocument[]>;
  featureDocuments: TaskNodeDocument[];
  selectedTask: Task | undefined;
  previewDocument: TaskNodeDocument | undefined;
  documentTarget: DocumentTarget | undefined;
  showTask: boolean;
  showFeatureEdit: boolean;
  syncPending: boolean;
  onSync: () => void;
  onCreateTask: () => void;
  onEditTask: (taskId: string) => void;
  onPreviewDocument: (document: TaskNodeDocument) => void;
  onAddDocument: (taskId: string, trigger: HTMLButtonElement) => void;
  onEditFeature: () => void;
  onCloseInspector: () => void;
  onClosePreview: () => void;
  onCloseDocumentDialog: () => void;
  onCloseTask: () => void;
  onCloseFeatureEdit: () => void;
  onFeatureDeleted: () => void;
}

function WorkspaceContent(props: WorkspaceContentProps) {
  // The server decides read-only, so the workspace never combines the
  // feature's own archived flag with the state of its project.
  const readOnly = props.feature.readOnly;
  return (
    <div className={readOnly ? "workspace is-archived" : "workspace"}>
      <FeatureWorkspaceHead props={props} readOnly={readOnly} />
      {readOnly && <ArchivedNotice project={props.project} />}
      <div className="workspace-body">
        <FeatureGraph
          tasks={props.tasks}
          dependencies={props.dependencies}
          pullRequests={props.pullRequests}
          documentsByTask={props.documentsByTask}
          onEditTask={props.onEditTask}
          onPreviewDocument={props.onPreviewDocument}
          onAddDocument={props.onAddDocument}
          onCreateTask={props.onCreateTask}
          readOnly={readOnly}
        />
        {props.selectedTask && (
          <TaskInspector
            task={props.selectedTask}
            tasks={props.tasks}
            pullRequest={props.pullRequests.get(props.selectedTask.id)}
            documents={props.documentsByTask.get(props.selectedTask.id) ?? []}
            readOnly={readOnly}
            onPreview={props.onPreviewDocument}
            onClose={props.onCloseInspector}
          />
        )}
      </div>
      <WorkspaceOverlays props={props} />
    </div>
  );
}

function FeatureWorkspaceHead({
  props,
  readOnly,
}: {
  props: WorkspaceContentProps;
  readOnly: boolean;
}) {
  const { t } = useTranslation();
  return (
    <header className="workspace-head">
      <div
        className="workspace-title"
        title={props.feature.description || t("workspace.noDescription")}
      >
        <h1>{props.feature.title}</h1>
        <CopyableIdentifier
          label={t("common.featureId")}
          value={props.feature.id}
          valueOnly
        />
        <p className="eyebrow">
          {t("workspace.eyebrow", { slug: props.feature.slug })}
          {props.project && (
            <>
              {" · "}
              <Link
                to="/projects/$projectId"
                params={{ projectId: props.project.id }}
                className="workspace-project-link"
              >
                {props.project.title}
              </Link>
            </>
          )}
        </p>
      </div>
      <div className="workspace-actions">
        <DocumentReferences
          parent={{ featureId: props.featureId }}
          documents={props.featureDocuments}
          onPreview={props.onPreviewDocument}
          readOnly={readOnly}
        />
        {!readOnly && (
          <IconButton
            icon={RefreshCw}
            label={
              props.syncPending
                ? t("workspace.syncing")
                : t("workspace.syncGithub")
            }
            variant="secondary"
            onClick={props.onSync}
            disabled={props.syncPending}
          />
        )}
        {!readOnly && (
          <IconButton
            icon={Plus}
            label={t("workspace.addTask")}
            variant="primary"
            onClick={props.onCreateTask}
          />
        )}
        <IconButton
          icon={Pencil}
          label={
            readOnly ? t("workspace.manageFeature") : t("workspace.editFeature")
          }
          variant="secondary"
          iconOnly
          onClick={props.onEditFeature}
        />
      </div>
    </header>
  );
}

// A feature can be read-only for two reasons, and the remedy differs: restore
// the feature, or activate the project it belongs to. The notice says which,
// and an archived project decides even when the feature is archived too:
// restoring the feature alone would leave it read-only.
function ArchivedNotice({ project }: { project: Project | undefined }) {
  const { t } = useTranslation();
  if (project?.archived)
    return (
      <div className="archived-notice" role="status">
        <strong>{t("workspace.projectArchivedLabel")}</strong>
        <span>
          {t("workspace.projectArchivedDetail", { title: project.title })}
        </span>
        <Link to="/projects/$projectId" params={{ projectId: project.id }}>
          {t("workspace.openProject")}
        </Link>
      </div>
    );
  return (
    <div className="archived-notice" role="status">
      <strong>{t("workspace.archivedLabel")}</strong>
      <span>{t("workspace.archivedDetail")}</span>
    </div>
  );
}

function WorkspaceOverlays({ props }: { props: WorkspaceContentProps }) {
  return (
    <>
      {props.previewDocument && (
        <MarkdownPreview
          key={props.previewDocument.id}
          document={props.previewDocument}
          onClose={props.onClosePreview}
        />
      )}
      {props.documentTarget && (
        <AddDocumentDialog
          taskId={props.documentTarget.taskId}
          trigger={props.documentTarget.trigger}
          onClose={props.onCloseDocumentDialog}
        />
      )}
      {props.showTask && (
        <CreateTaskDialog
          featureId={props.featureId}
          onClose={props.onCloseTask}
        />
      )}
      {props.showFeatureEdit && (
        <EditFeatureDialog
          feature={props.feature}
          projects={props.projects}
          onClose={props.onCloseFeatureEdit}
          onDeleted={props.onFeatureDeleted}
        />
      )}
    </>
  );
}
