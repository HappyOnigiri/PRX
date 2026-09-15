import { Link, useNavigate, useParams } from "@tanstack/react-router";
import {
  ArrowLeft,
  ClipboardList,
  Pencil,
  Plus,
  RefreshCw,
  Search,
  X,
} from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
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
import { BatchPromptDialog } from "./BatchPromptDialog";
import { CopyableIdentifier } from "./CopyableIdentifier";
import { CreateTaskDialog } from "./CreateTaskDialog";
import type { PendingDependency } from "./dependencyGraph";
import { DocumentDialog } from "./DocumentDialog";
import { DocumentReferences } from "./DocumentReferences";
import { EditFeatureDialog } from "./EditFeatureDialog";
import { EntityIcon } from "./EntityIcon";
import { FeatureGraph } from "./FeatureGraph";
import { matchesGraphSearch } from "./graphSearch";
import { IconButton } from "./IconButton";
import { MarkdownPreview } from "./MarkdownPreview";
import { StatusBadge } from "./StatusBadge";
import { TaskInspector } from "./TaskInspector";
import { type TaskNodeDocument } from "./TaskNode";
import { useCloseOnEscape } from "./useCloseOnEscape";
import {
  emptyHiddenDependencies,
  hideTasks,
  isFinishedTask,
  type HiddenDependencies,
  type VisibleGraph,
} from "./visibleGraph";

interface DocumentTarget {
  taskId: string;
  trigger: HTMLButtonElement;
}

// 作成ダイアログは、グラフの空白ドロップで開くときだけ依存の相手を伴う。
interface TaskDraft {
  dependency?: PendingDependency;
}

export function FeatureWorkspace() {
  const { featureId } = useParams({ from: "/features/$featureId" });
  const navigate = useNavigate();
  const snapshot = useSnapshot();
  const [selected, setSelected] = useState<string>();
  const [taskDraft, setTaskDraft] = useState<TaskDraft>();
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
  const search = useGraphSearch();
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
  const visible = useVisibleGraph(
    tasks,
    dependencies,
    pullRequests,
    hideCompleted,
    search.query,
  );
  // Cmd/Ctrl+F はブラウザの検索を奪うので、キャンバスが前面にある間だけ受ける。
  // 手前でダイアログが開いているときは、その入力欄の検索を邪魔しない。
  const overlayOpen =
    taskDraft !== undefined ||
    previewDocument !== undefined ||
    documentTarget !== undefined ||
    showFeatureEdit ||
    showBatchPrompt;
  useGraphSearchShortcut(Boolean(feature) && !overlayOpen, search.open);
  // 開いている作成ダイアログは開き直しても置き換えない。背景のツールバーへ
  // フォーカスが届くので、指定済みの依存と入力を消さずに残す。
  const openTaskDialog = useCallback((dependency?: PendingDependency) => {
    setTaskDraft(
      (current) => current ?? { ...(dependency ? { dependency } : {}) },
    );
  }, []);
  const openDocumentDialog = useCallback(
    (taskId: string, trigger: HTMLButtonElement) => {
      setDocumentTarget({ taskId, trigger });
    },
    [],
  );
  const sync = useDomainMutation((id: string) => mutations.sync(id));

  if (snapshot.isPending) return <WorkspaceLoading />;
  if (!feature || !data) return <WorkspaceMissing />;

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
      search={search}
      pullRequests={pullRequests}
      documentsByTask={documentsByTask}
      featureDocuments={featureDocuments}
      selectedTask={selectedTask}
      previewDocument={previewDocument}
      documentTarget={documentTarget}
      taskDraft={taskDraft}
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
        setTaskDraft(undefined);
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

function WorkspaceLoading() {
  const { t } = useTranslation();
  return (
    <div className="state-message">
      <div className="spinner" />
      <h1>{t("workspace.loading")}</h1>
    </div>
  );
}

function WorkspaceMissing() {
  const { t } = useTranslation();
  const navigate = useNavigate();
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
// 完了済みと検索の不一致は 1 つの述語に束ね、隠す処理は 1 度だけ通す。
function useVisibleGraph(
  tasks: Task[],
  dependencies: Dependency[],
  pullRequests: Map<string, PullRequest>,
  hideCompleted: boolean,
  query: string,
): VisibleGraph {
  return useMemo(() => {
    if (!hideCompleted && query === "")
      return {
        tasks,
        dependencies,
        hiddenDependencies: emptyHiddenDependencies,
      };
    return hideTasks(
      tasks,
      dependencies,
      (task) =>
        (hideCompleted && isFinishedTask(task)) ||
        !matchesGraphSearch(task, pullRequests.get(task.id), query),
    );
  }, [tasks, dependencies, pullRequests, hideCompleted, query]);
}

// 検索語は URL にもブラウザローカルにも残さない。1 回の閲覧の中で使う一時的な
// 操作だからである。docs/design/webui.md を参照。
interface GraphSearch {
  opened: boolean;
  input: string;
  query: string;
  inputRef: React.RefObject<HTMLInputElement | null>;
  open: () => void;
  close: () => void;
  change: (value: string) => void;
}

function useGraphSearch(): GraphSearch {
  const [opened, setOpened] = useState(false);
  const [input, setInput] = useState("");
  const inputRef = useRef<HTMLInputElement>(null);
  const settled = useDebounced(input, 200);
  const open = useCallback(() => {
    setOpened(true);
    // 既に開いているときは打ち直せるよう、入力欄へ戻して全選択する。
    const field = inputRef.current;
    if (!field) return;
    field.focus();
    field.select();
  }, []);
  const close = useCallback(() => {
    setOpened(false);
    setInput("");
  }, []);
  return {
    opened,
    input,
    query: opened ? settled.trim() : "",
    inputRef,
    open,
    close,
    change: setInput,
  };
}

// ELK のレイアウトは Web Worker で走るので、打鍵ごとには絞り込まない。入力欄の
// 値は即時に映し、グラフへ渡す語だけを遅らせる。
function useDebounced(value: string, delay: number): string {
  const [settled, setSettled] = useState(value);
  useEffect(() => {
    const timer = window.setTimeout(() => {
      setSettled(value);
    }, delay);
    return () => {
      window.clearTimeout(timer);
    };
  }, [value, delay]);
  return settled;
}

function useGraphSearchShortcut(enabled: boolean, open: () => void) {
  const latest = useRef(open);
  useEffect(() => {
    latest.current = open;
  });
  useEffect(() => {
    if (!enabled) return;
    const handle = (event: KeyboardEvent) => {
      if (event.key !== "f" || event.altKey || event.shiftKey) return;
      if (!event.metaKey && !event.ctrlKey) return;
      event.preventDefault();
      latest.current();
    };
    window.addEventListener("keydown", handle);
    return () => {
      window.removeEventListener("keydown", handle);
    };
  }, [enabled]);
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
  search: GraphSearch;
  pullRequests: Map<string, PullRequest>;
  documentsByTask: Map<string, TaskNodeDocument[]>;
  featureDocuments: TaskNodeDocument[];
  selectedTask: Task | undefined;
  previewDocument: TaskNodeDocument | undefined;
  documentTarget: DocumentTarget | undefined;
  taskDraft: TaskDraft | undefined;
  showFeatureEdit: boolean;
  showBatchPrompt: boolean;
  syncPending: boolean;
  onSync: () => void;
  onCreateTask: (dependency?: PendingDependency) => void;
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
          searching={props.search.query !== ""}
          pullRequests={props.pullRequests}
          documentsByTask={props.documentsByTask}
          onEditTask={props.onEditTask}
          onPreviewDocument={props.onPreviewDocument}
          onAddDocument={props.onAddDocument}
          onCreateTask={props.onCreateTask}
          taskLabelAppearances={props.feature.taskLabelAppearances}
          readOnly={readOnly}
        />
        {props.selectedTask && (
          <TaskInspector
            key={props.selectedTask.id}
            task={props.selectedTask}
            tasks={props.tasks}
            pullRequest={props.pullRequests.get(props.selectedTask.id)}
            documents={props.documentsByTask.get(props.selectedTask.id) ?? []}
            appearances={props.feature.taskLabelAppearances}
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
        <GraphSearchControl
          search={props.search}
          matched={props.visibleTasks.length}
          total={props.tasks.length}
        />
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
            label={t("workspace.refresh")}
            variant="secondary"
            busy={props.syncPending}
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
            onClick={() => {
              props.onCreateTask();
            }}
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

// ショートカットだけでは機能があることが伝わらないので、ツールバーにも入口を
// 置く。Cmd/Ctrl+F はその加速手段として添える。
function GraphSearchControl({
  search,
  matched,
  total,
}: {
  search: GraphSearch;
  matched: number;
  total: number;
}) {
  const { t } = useTranslation();
  const buttonRef = useRef<HTMLButtonElement>(null);
  // 閉じた後の焦点は、検索を開いたアイコンへ戻す。
  const wasOpened = useRef(search.opened);
  useEffect(() => {
    if (wasOpened.current && !search.opened) buttonRef.current?.focus();
    wasOpened.current = search.opened;
  }, [search.opened]);
  if (!search.opened)
    return (
      <IconButton
        icon={Search}
        label={t("workspace.searchTasks")}
        variant="secondary"
        iconOnly
        ref={buttonRef}
        onClick={search.open}
      />
    );
  return <GraphSearchField search={search} matched={matched} total={total} />;
}

function GraphSearchField({
  search,
  matched,
  total,
}: {
  search: GraphSearch;
  matched: number;
  total: number;
}) {
  const { t } = useTranslation();
  const { close, inputRef } = search;
  // 検索窓は Escape の重なりに乗せる。手前でダイアログが開いていれば、規約
  // どおりそちらが先に閉じる。
  useCloseOnEscape(close);
  useEffect(() => {
    inputRef.current?.focus();
  }, [inputRef]);
  return (
    <div className="graph-search" role="search">
      <Search aria-hidden="true" focusable="false" size={14} />
      <input
        ref={inputRef}
        type="text"
        className="graph-search-input"
        value={search.input}
        aria-label={t("workspace.searchLabel")}
        aria-describedby="graph-search-count"
        placeholder={t("workspace.searchPlaceholder")}
        autoComplete="off"
        onChange={(event) => {
          search.change(event.currentTarget.value);
        }}
      />
      <span
        className="graph-search-count"
        id="graph-search-count"
        aria-live="polite"
      >
        {t("workspace.searchMatchCount", { matched, total })}
      </span>
      <IconButton
        icon={X}
        label={t("workspace.searchClose")}
        variant="quiet"
        size="compact"
        iconOnly
        onClick={close}
      />
    </div>
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

function TaskCreateOverlay({
  draft,
  featureId,
  tasks,
  onClose,
}: {
  draft: TaskDraft;
  featureId: string;
  tasks: Task[];
  onClose: () => void;
}) {
  const dependency = draft.dependency;
  if (!dependency)
    return <CreateTaskDialog featureId={featureId} onClose={onClose} />;
  const title = tasks.find((task) => task.id === dependency.taskId)?.title;
  return (
    <CreateTaskDialog
      featureId={featureId}
      onClose={onClose}
      dependency={{ value: dependency, title: title ?? dependency.taskId }}
    />
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
        <DocumentDialog
          mode="add"
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
      {props.taskDraft && (
        <TaskCreateOverlay
          draft={props.taskDraft}
          featureId={props.featureId}
          tasks={props.tasks}
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
