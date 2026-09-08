import { Link, useNavigate, useParams } from "@tanstack/react-router";
import {
  ArrowLeft,
  ClipboardList,
  Pencil,
  Plus,
  RefreshCw,
} from "lucide-react";
import { useCallback, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { mutations } from "../api";
import type {
  Dependency,
  Feature,
  FeatureStatus,
  Project,
  PullRequest,
  Snapshot,
  Task,
} from "../gen/prx/v1/prx_pb";
import { useDomainMutation, useSnapshot } from "../hooks";
import { featureStatusLabel, featureStatusToken } from "../i18n/domain";
import {
  readHideCompletedTasks,
  writeHideCompletedTasks,
} from "../i18n/settings";
import { AddDocumentDialog } from "./AddDocumentDialog";
import { BatchPromptDialog } from "./BatchPromptDialog";
import {
  emptyHiddenDependencies,
  hideFinishedTasks,
  type HiddenDependencies,
} from "./completedTasks";
import { CopyableIdentifier } from "./CopyableIdentifier";
import { CreateTaskDialog } from "./CreateTaskDialog";
import { DocumentReferences } from "./DocumentReferences";
import { EditFeatureDialog } from "./EditFeatureDialog";
import { EntityIcon } from "./EntityIcon";
import { FeatureGraph } from "./FeatureGraph";
import { IconButton } from "./IconButton";
import { MarkdownPreview } from "./MarkdownPreview";
import { StatusBadge } from "./StatusBadge";
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
  const [showBatchPrompt, setShowBatchPrompt] = useState(false);
  const [previewDocument, setPreviewDocument] = useState<TaskNodeDocument>();
  const [documentTarget, setDocumentTarget] = useState<DocumentTarget>();
  // 完了タスクの非表示は前回グラフをどう読んだかを表すので、次回の訪問時に
  // ブラウザローカルの設定から復元する。
  const [hideCompleted, setHideCompleted] = useState(readHideCompletedTasks);
  const changeHideCompleted = useCallback((hide: boolean) => {
    setHideCompleted(hide);
    writeHideCompletedTasks(hide);
  }, []);
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
  const visible = useVisibleGraph(tasks, dependencies, hideCompleted);
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

  // インスペクタはキャンバスに従う。読み手が隠したタスクは、隣のパネルに
  // 開いたまま残らずノードと一緒に画面から消える。
  const selectedTask = visible.tasks.find((task) => task.id === selected);
  return (
    <WorkspaceContent
      feature={feature}
      featureId={featureId}
      project={project}
      projects={data.projects}
      tasks={tasks}
      visibleTasks={visible.tasks}
      visibleDependencies={visible.dependencies}
      hiddenDependencies={visible.hiddenDependencies}
      hiddenTaskCount={tasks.length - visible.tasks.length}
      hideCompleted={hideCompleted}
      onHideCompletedChange={changeHideCompleted}
      pullRequests={pullRequests}
      documentsByTask={documentsByTask}
      featureDocuments={featureDocuments}
      selectedTask={selectedTask}
      previewDocument={previewDocument}
      documentTarget={documentTarget}
      showTask={showTask}
      showFeatureEdit={showFeatureEdit}
      showBatchPrompt={showBatchPrompt}
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
      onCopyBatchPrompt={() => {
        setShowBatchPrompt(true);
      }}
      onCloseBatchPrompt={() => {
        setShowBatchPrompt(false);
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
        // 削除後はその feature へ辿れたリストへ戻る。読み取り専用なら、そもそも
        // 現れない概要ではなく、所有プロジェクトのアーカイブタブ。
        void navigate(deletedFeatureDestination(feature));
      }}
    />
  );
}

function deletedFeatureDestination(feature: Feature) {
  if (!feature.readOnly) return { to: "/" } as const;
  return {
    to: "/projects/$projectId",
    params: { projectId: feature.projectId },
    search: { features: "archived" },
  } as const;
}

// ワークスペースはスナップショットを 1 つの feature に絞り込む。memo をここに
// まとめることで、コンポーネント本体は描画と状態に専念できる。
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
  // feature に属さないドキュメントは ID が欠けるのではなく空文字になるので、
  // 比較では空の値を弾く必要がある。
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

// レイアウトは渡されたものの同一性を見るので、フィルタを使わないときは複製では
// なくスナップショットが作った配列そのものを返す必要がある。
function useVisibleGraph(
  tasks: Task[],
  dependencies: Dependency[],
  hideCompleted: boolean,
) {
  return useMemo(
    () =>
      hideCompleted
        ? hideFinishedTasks(tasks, dependencies)
        : { tasks, dependencies, hiddenDependencies: emptyHiddenDependencies },
    [tasks, dependencies, hideCompleted],
  );
}

interface WorkspaceContentProps {
  feature: Feature;
  featureId: string;
  project: Project | undefined;
  projects: Project[];
  tasks: Task[];
  visibleTasks: Task[];
  visibleDependencies: Dependency[];
  hiddenDependencies: Map<string, HiddenDependencies>;
  hiddenTaskCount: number;
  hideCompleted: boolean;
  onHideCompletedChange: (hide: boolean) => void;
  pullRequests: Map<string, PullRequest>;
  documentsByTask: Map<string, TaskNodeDocument[]>;
  featureDocuments: TaskNodeDocument[];
  selectedTask: Task | undefined;
  previewDocument: TaskNodeDocument | undefined;
  documentTarget: DocumentTarget | undefined;
  showTask: boolean;
  showFeatureEdit: boolean;
  showBatchPrompt: boolean;
  syncPending: boolean;
  onSync: () => void;
  onCreateTask: () => void;
  onEditTask: (taskId: string) => void;
  onPreviewDocument: (document: TaskNodeDocument) => void;
  onAddDocument: (taskId: string, trigger: HTMLButtonElement) => void;
  onEditFeature: () => void;
  onCopyBatchPrompt: () => void;
  onCloseBatchPrompt: () => void;
  onCloseInspector: () => void;
  onClosePreview: () => void;
  onCloseDocumentDialog: () => void;
  onCloseTask: () => void;
  onCloseFeatureEdit: () => void;
  onFeatureDeleted: () => void;
}

function WorkspaceContent(props: WorkspaceContentProps) {
  // 読み取り専用かはサーバーが決めるので、ワークスペースは feature 自身の
  // アーカイブフラグとプロジェクトの状態を組み合わせない。
  const readOnly = props.feature.readOnly;
  return (
    <div className={readOnly ? "workspace is-archived" : "workspace"}>
      <FeatureWorkspaceHead props={props} readOnly={readOnly} />
      {readOnly && <ArchivedNotice project={props.project} />}
      <div className="workspace-body">
        <FeatureGraph
          tasks={props.visibleTasks}
          dependencies={props.visibleDependencies}
          hiddenDependencies={props.hiddenDependencies}
          hiddenTaskCount={props.hiddenTaskCount}
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
            key={props.selectedTask.id}
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
        <FeatureStatusBadge status={props.feature.displayStatus} />
        <h1>
          <EntityIcon kind="feature" size={17} />
          {props.feature.title}
        </h1>
        <CopyableIdentifier
          label={t("common.featureId")}
          value={props.feature.id}
          valueOnly
        />
        {props.project && (
          <p className="eyebrow">
            <Link
              to="/projects/$projectId"
              params={{ projectId: props.project.id }}
              search={{ features: "active" }}
              className="workspace-project-link"
            >
              {props.project.title}
            </Link>
          </p>
        )}
      </div>
      <div className="workspace-actions">
        <HideCompletedToggle
          checked={props.hideCompleted}
          onChange={props.onHideCompletedChange}
        />
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
        {/* アーカイブ済みの feature でもコピーは使える。エージェントへの受け渡しは
            PRX を変更せず読むだけだから。 */}
        <IconButton
          icon={ClipboardList}
          label={t("batchPrompt.open")}
          variant="secondary"
          onClick={props.onCopyBatchPrompt}
        />
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

// このフィルタは 1 つのグラフを読む間に切り替えるもので次回の訪問では戻るため、
// 永続設定が集まる設定ダイアログではなく、対象のキャンバスに置く。
function HideCompletedToggle({
  checked,
  onChange,
}: {
  checked: boolean;
  onChange: (hide: boolean) => void;
}) {
  const { t } = useTranslation();
  return (
    <label className="hide-completed-toggle">
      {/* 狭い画面では文言が省かれるので、隣のテキストに頼らず
          コントロール自身が名前を持つ。 */}
      <input
        type="checkbox"
        role="switch"
        aria-label={t("workspace.hideCompleted")}
        title={t("workspace.hideCompleted")}
        checked={checked}
        onChange={(event) => {
          onChange(event.currentTarget.checked);
        }}
      />
      <span>{t("workspace.hideCompleted")}</span>
    </label>
  );
}

// ヘッダーはサーバーが導出した状態を、feature 一覧と同じ値で表示する。自動の
// ままの feature もどちらの場所でも同じに読める。
function FeatureStatusBadge({ status }: { status: FeatureStatus }) {
  const { t } = useTranslation();
  return (
    <StatusBadge
      className={`status-${featureStatusToken(status)}`}
      label={featureStatusLabel(status, t)}
      title={t("workspace.featureStatus")}
    />
  );
}

// 読み取り専用の理由は 2 つあり、対処も違う。feature の復元か、プロジェクトの
// 再開。feature もアーカイブ済みならプロジェクト側が優先される。feature だけ
// 戻しても何も変わらないため。
function ArchivedNotice({ project }: { project: Project | undefined }) {
  const { t } = useTranslation();
  if (project?.archived)
    return (
      <div className="archived-notice" role="status">
        <strong>{t("workspace.projectArchivedLabel")}</strong>
        <span>
          {t("workspace.projectArchivedDetail", { title: project.title })}
        </span>
        <Link
          to="/projects/$projectId"
          params={{ projectId: project.id }}
          search={{ features: "archived" }}
        >
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
      {props.showBatchPrompt && (
        <BatchPromptDialog
          featureId={props.featureId}
          tasks={props.tasks}
          onClose={props.onCloseBatchPrompt}
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
