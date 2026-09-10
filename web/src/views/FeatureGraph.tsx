import {
  Background,
  BackgroundVariant,
  Controls,
  MarkerType,
  ReactFlow,
  type AriaLabelConfig,
  type Connection,
  type Edge,
  type EdgeChange,
  type EdgeMouseHandler,
  type FinalConnectionState,
  type HandleType,
  type OnEdgesChange,
  type OnEdgesDelete,
  type OnMove,
  type OnReconnect,
  type ReactFlowInstance,
} from "@xyflow/react";
import type { TFunction } from "i18next";
import { EyeOff, Plus, RotateCcw, TriangleAlert } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { mutations } from "../api";
import type { Dependency, PullRequest, Task } from "../gen/prx/v1/prx_pb";
import { useDomainMutation } from "../hooks";
import {
  maxGraphZoom,
  minGraphZoom,
  readGraphZoom,
  writeGraphZoom,
} from "../i18n/settings";
import {
  emptyHiddenDependencies,
  type HiddenDependencies,
} from "./completedTasks";
import { DependencyEdge } from "./DependencyEdge";
import {
  dependencyEdgeId,
  type DependencyEdgeRoute,
  type DependencyFlowEdge,
  type PendingDependency,
} from "./dependencyGraph";
import { IconButton } from "./IconButton";
import { MutationError } from "./MutationError";
import { TaskNode, type TaskFlowNode, type TaskNodeDocument } from "./TaskNode";
import { useGraphLayout } from "./useGraphLayout";

const nodeTypes = { task: TaskNode };
const edgeTypes = { dependency: DependencyEdge };

// ノードとして扱われるのはハンドルの近傍だけなので、カード本体へのドロップが
// 空白へのドロップと混ざらないよう、ドロップ位置が矩形に入るかを自分で見る。
function droppedOnNode(
  event: MouseEvent | TouchEvent,
  flow: ReactFlowInstance<TaskFlowNode, DependencyFlowEdge> | undefined,
): boolean {
  if (!flow) return false;
  const point = "changedTouches" in event ? event.changedTouches[0] : event;
  if (!point) return false;
  const { x, y } = flow.screenToFlowPosition({
    x: point.clientX,
    y: point.clientY,
  });
  return flow.getNodes().some((node) => {
    const width = node.measured?.width ?? node.width ?? 0;
    const height = node.measured?.height ?? node.height ?? 0;
    return (
      x >= node.position.x &&
      x <= node.position.x + width &&
      y >= node.position.y &&
      y <= node.position.y + height
    );
  });
}

function useDependencyConnections(
  readOnly: boolean,
  onCreateTask: (dependency?: PendingDependency) => void,
  flow: ReactFlowInstance<TaskFlowNode, DependencyFlowEdge> | undefined,
) {
  const addDependency = useDomainMutation(
    ({ blocker, blocked }: { blocker: string; blocked: string }) =>
      mutations.addDependency(blocker, blocked),
  );
  const removeDependency = useDomainMutation(
    ({ blocker, blocked }: { blocker: string; blocked: string }) =>
      mutations.removeDependency(blocker, blocked),
  );
  const [detaching, setDetaching] = useState<
    { blocker: string; blocked: string } | undefined
  >();
  const [connecting, setConnecting] = useState(false);
  // 既存エッジの掴み直しでも onConnectEnd が先に呼ばれるので、空白ドロップが
  // 依存の解除なのか新規作成なのかをここで見分ける。
  const reconnecting = useRef(false);
  const pending = addDependency.isPending || removeDependency.isPending;
  const onConnect = useCallback(
    ({ source, target }: Connection) => {
      if (readOnly || pending || !source || !target) return;
      addDependency.mutate({ blocker: source, blocked: target });
    },
    [addDependency, pending, readOnly],
  );
  const onConnectStart = useCallback(() => {
    if (!readOnly) setConnecting(true);
  }, [readOnly]);
  // 空白へのドロップは、その依存の相手がまだ居ないという意思表示として扱い、
  // 依存付きのタスク作成へつなぐ。
  const onConnectEnd = useCallback(
    (event: MouseEvent | TouchEvent, connectionState: FinalConnectionState) => {
      setConnecting(false);
      if (readOnly || pending || reconnecting.current) return;
      if (connectionState.isValid === true || connectionState.toNode) return;
      if (droppedOnNode(event, flow)) return;
      const fromNode = connectionState.fromNode;
      const handleType = connectionState.fromHandle?.type;
      if (!fromNode || (handleType !== "source" && handleType !== "target"))
        return;
      onCreateTask({
        taskId: fromNode.id,
        direction: handleType === "source" ? "blockedBy" : "blocks",
      });
    },
    [flow, onCreateTask, pending, readOnly],
  );
  const remove = useCallback(
    (edge: Pick<Edge, "source" | "target">) => {
      if (readOnly || removeDependency.isPending) return;
      removeDependency.mutate({ blocker: edge.source, blocked: edge.target });
    },
    [readOnly, removeDependency],
  );
  const onEdgesDelete = useCallback<OnEdgesDelete<DependencyFlowEdge>>(
    (edges) => {
      for (const edge of edges) remove(edge);
    },
    [remove],
  );
  const onReconnect = useCallback<OnReconnect<DependencyFlowEdge>>(() => {
    setDetaching(undefined);
  }, []);
  const onReconnectStart = useCallback(
    (_event: React.MouseEvent, edge: DependencyFlowEdge) => {
      void _event;
      reconnecting.current = true;
      if (!readOnly)
        setDetaching({ blocker: edge.source, blocked: edge.target });
    },
    [readOnly],
  );
  const onReconnectEnd = useCallback(
    (
      _event: MouseEvent | TouchEvent,
      edge: DependencyFlowEdge,
      _handleType: HandleType,
      connectionState: FinalConnectionState,
    ) => {
      void _event;
      void _handleType;
      reconnecting.current = false;
      setDetaching(undefined);
      if (connectionState.isValid === true) return;
      remove(edge);
    },
    [remove],
  );

  return {
    adding: addDependency.isPending,
    connecting,
    detaching,
    error: removeDependency.error ?? addDependency.error,
    onConnect,
    onConnectEnd,
    onConnectStart,
    onEdgesDelete,
    onReconnect,
    onReconnectEnd,
    onReconnectStart,
    pending,
    remove,
    removing: removeDependency.isPending,
  };
}

function buildDependencyEdges({
  dependencies,
  edgeRoutes,
  pending,
  readOnly,
  remove,
  selectedId,
  taskTitle,
  t,
}: {
  dependencies: Dependency[];
  edgeRoutes: Map<string, DependencyEdgeRoute>;
  pending: boolean;
  readOnly: boolean;
  remove: (edge: Pick<Edge, "source" | "target">) => void;
  selectedId: string | undefined;
  taskTitle: (taskId: string) => string | undefined;
  t: TFunction;
}): DependencyFlowEdge[] {
  return dependencies.map((dependency) => {
    const id = dependencyEdgeId(
      dependency.blockerTaskId,
      dependency.blockedTaskId,
    );
    const route = edgeRoutes.get(id);
    const blocker =
      taskTitle(dependency.blockerTaskId) ?? dependency.blockerTaskId;
    const blocked =
      taskTitle(dependency.blockedTaskId) ?? dependency.blockedTaskId;
    return {
      id,
      source: dependency.blockerTaskId,
      sourceHandle: route?.sourcePortId ?? null,
      target: dependency.blockedTaskId,
      targetHandle: route?.targetPortId ?? null,
      type: "dependency",
      data: {
        disabled: pending,
        label: t("workspace.flow.dependencyLabel", { blocker, blocked }),
        onRemove: () => {
          remove({
            source: dependency.blockerTaskId,
            target: dependency.blockedTaskId,
          });
        },
        readOnly,
        removeLabel: t("workspace.flow.removeDependency", {
          blocker,
          blocked,
        }),
        route,
      },
      deletable: !readOnly && !pending,
      interactionWidth: 24,
      markerEnd: { type: MarkerType.ArrowClosed },
      className: "dependency-edge",
      reconnectable: readOnly || pending ? false : true,
      selected: id === selectedId,
      ariaLabel: t("workspace.flow.dependencyEdge", { blocker, blocked }),
    };
  });
}

function useTaskTitle(tasks: Task[]) {
  const titlesByTask = useMemo(
    () => new Map(tasks.map((task) => [task.id, task.title])),
    [tasks],
  );
  return useCallback(
    (taskId: string) => titlesByTask.get(taskId),
    [titlesByTask],
  );
}

// React Flow はエッジ配列を参照で比較するので、描画のたびに新しい配列を渡すと
// 接続の索引が作り直され、ストアの利用側がすべて再描画される。
function useDependencyEdges(options: {
  dependencies: Dependency[];
  edgeRoutes: Map<string, DependencyEdgeRoute>;
  pending: boolean;
  readOnly: boolean;
  remove: (edge: Pick<Edge, "source" | "target">) => void;
  selectedId: string | undefined;
  taskTitle: (taskId: string) => string | undefined;
  t: TFunction;
}) {
  const {
    dependencies,
    edgeRoutes,
    pending,
    readOnly,
    remove,
    selectedId,
    taskTitle,
    t,
  } = options;
  return useMemo(
    () =>
      buildDependencyEdges({
        dependencies,
        edgeRoutes,
        pending,
        readOnly,
        remove,
        selectedId,
        taskTitle,
        t,
      }),
    [
      dependencies,
      edgeRoutes,
      pending,
      readOnly,
      remove,
      selectedId,
      taskTitle,
      t,
    ],
  );
}

// React Flow は端点のポートを、それを持つノードの確定から 1 フレーム後に測る。
// このフレームを待つことで、まだ存在しないハンドルを指してエッジが
// キャンバスから落ちるのを防ぐ。
function useMeasuredEdgeRoutes(edgeRoutes: Map<string, DependencyEdgeRoute>) {
  const [measured, setMeasured] = useState<Map<string, DependencyEdgeRoute>>(
    () => new Map(),
  );
  useEffect(() => {
    const frame = requestAnimationFrame(() => {
      setMeasured(edgeRoutes);
    });
    return () => {
      cancelAnimationFrame(frame);
    };
  }, [edgeRoutes]);
  return measured;
}

function useDependencySelection(dependencies: Dependency[]) {
  const [requestedId, setSelectedId] = useState<string>();
  const stillPresent = dependencies.some(
    (dependency) =>
      dependencyEdgeId(dependency.blockerTaskId, dependency.blockedTaskId) ===
      requestedId,
  );
  const selectedId = stillPresent ? requestedId : undefined;
  // ID を捨てることで、再追加された依存がユーザーの操作なしに選択済みとして
  // 現れるのを防ぐ。
  if (requestedId !== undefined && !stillPresent) setSelectedId(undefined);
  const clear = useCallback(() => {
    setSelectedId(undefined);
  }, []);
  const select = useCallback<EdgeMouseHandler<DependencyFlowEdge>>(
    (_event, edge) => {
      setSelectedId(edge.id);
    },
    [],
  );
  // エッジは制御下にあるため React Flow 自身の選択更新は捨てられる。ここで
  // 変更を適用して初めて、フォーカス中のエッジでの Enter や Space が
  // ツールバーと Delete キーに届く。
  const applyChanges = useCallback(
    (changes: EdgeChange<DependencyFlowEdge>[]) => {
      for (const change of changes) {
        if (change.type !== "select") continue;
        setSelectedId((current) =>
          change.selected
            ? change.id
            : current === change.id
              ? undefined
              : current,
        );
      }
    },
    [],
  );
  return { applyChanges, clear, select, selectedId };
}

interface FeatureGraphProps {
  tasks: Task[];
  dependencies: Dependency[];
  pullRequests: Map<string, PullRequest>;
  documentsByTask: Map<string, TaskNodeDocument[]>;
  hiddenDependencies?: Map<string, HiddenDependencies>;
  hiddenTaskCount?: number;
  onEditTask: (taskId: string) => void;
  onPreviewDocument: (document: TaskNodeDocument) => void;
  onAddDocument?: (taskId: string, trigger: HTMLButtonElement) => void;
  onCreateTask: (dependency?: PendingDependency) => void;
  readOnly?: boolean;
}

export function FeatureGraph({
  tasks,
  dependencies,
  pullRequests,
  documentsByTask,
  hiddenDependencies = emptyHiddenDependencies,
  hiddenTaskCount = 0,
  onEditTask,
  onPreviewDocument,
  onAddDocument,
  onCreateTask,
  readOnly = false,
}: FeatureGraphProps) {
  const { t } = useTranslation();
  const [flow, setFlow] =
    useState<ReactFlowInstance<TaskFlowNode, DependencyFlowEdge>>();
  const [initialGraphZoom] = useState(readGraphZoom);
  const graphZoom = useRef(initialGraphZoom);
  const connections = useDependencyConnections(readOnly, onCreateTask, flow);
  const selection = useDependencySelection(dependencies);
  const taskTitle = useTaskTitle(tasks);
  const { edgeRoutes, nodes, layoutError, layoutPending, retryLayout } =
    useGraphLayout({
      tasks,
      dependencies,
      pullRequests,
      documentsByTask,
      hiddenDependencies,
      onEditTask,
      onPreviewDocument,
      ...(onAddDocument ? { onAddDocument } : {}),
      readOnly,
    });
  const measuredRoutes = useMeasuredEdgeRoutes(edgeRoutes);
  const edges = useDependencyEdges({
    dependencies,
    edgeRoutes: measuredRoutes,
    pending: connections.pending,
    readOnly,
    remove: connections.remove,
    selectedId: selection.selectedId,
    taskTitle,
    t,
  });
  const ariaLabelConfig = useMemo(
    () => ({
      "node.a11yDescription.default": t("workspace.flow.nodeDescription"),
      "node.a11yDescription.keyboardDisabled": t(
        "workspace.flow.keyboardDisabled",
      ),
      "edge.a11yDescription.default": t("workspace.flow.edgeDescription"),
      "controls.ariaLabel": t("workspace.flow.controls"),
      "controls.zoomIn.ariaLabel": t("workspace.flow.zoomIn"),
      "controls.zoomOut.ariaLabel": t("workspace.flow.zoomOut"),
      "controls.fitView.ariaLabel": t("workspace.flow.fitView"),
      "handle.ariaLabel": t("workspace.flow.handle"),
    }),
    [t],
  );

  useEffect(() => {
    if (nodes.length && flow) {
      const bounds = flow.getNodesBounds(nodes);
      void flow.setCenter(
        bounds.x + bounds.width / 2,
        bounds.y + bounds.height / 2,
        {
          zoom: graphZoom.current,
          duration: window.matchMedia("(prefers-reduced-motion: reduce)")
            .matches
            ? 0
            : 260,
        },
      );
    }
  }, [nodes, flow]);

  return (
    <>
      <GraphStatusNotice
        addingDependency={connections.adding}
        connecting={connections.connecting}
        detachingDependency={connections.detaching}
        removingDependency={connections.removing}
        readOnly={readOnly}
        taskTitle={taskTitle}
      />
      <GraphCanvas
        tasks={tasks}
        hiddenTaskCount={hiddenTaskCount}
        nodes={nodes}
        edges={edges}
        initialGraphZoom={initialGraphZoom}
        onInit={setFlow}
        onConnect={connections.onConnect}
        onConnectStart={connections.onConnectStart}
        onConnectEnd={connections.onConnectEnd}
        onEdgesChange={selection.applyChanges}
        onEdgesDelete={connections.onEdgesDelete}
        onEdgeClick={selection.select}
        onNodeClick={selection.clear}
        onPaneClick={selection.clear}
        onReconnect={connections.onReconnect}
        onReconnectStart={connections.onReconnectStart}
        onReconnectEnd={connections.onReconnectEnd}
        onMoveEnd={(_, viewport) => {
          graphZoom.current = viewport.zoom;
          writeGraphZoom(viewport.zoom);
        }}
        connectionPending={connections.pending}
        layoutPending={layoutPending}
        connectionError={connections.error}
        taskTitle={taskTitle}
        layoutError={layoutError}
        retryLayout={retryLayout}
        onCreateTask={onCreateTask}
        ariaLabelConfig={ariaLabelConfig}
        readOnly={readOnly}
      />
    </>
  );
}

interface GraphCanvasProps {
  tasks: Task[];
  hiddenTaskCount: number;
  nodes: TaskFlowNode[];
  edges: DependencyFlowEdge[];
  initialGraphZoom: number;
  onInit: (
    instance: ReactFlowInstance<TaskFlowNode, DependencyFlowEdge>,
  ) => void;
  onConnect: (connection: Connection) => void;
  onConnectStart: () => void;
  onConnectEnd: (
    event: MouseEvent | TouchEvent,
    connectionState: FinalConnectionState,
  ) => void;
  onEdgesChange: OnEdgesChange<DependencyFlowEdge>;
  onEdgesDelete: OnEdgesDelete<DependencyFlowEdge>;
  onEdgeClick: EdgeMouseHandler<DependencyFlowEdge>;
  onNodeClick: () => void;
  onPaneClick: () => void;
  onReconnect: OnReconnect<DependencyFlowEdge>;
  onReconnectStart: (
    event: React.MouseEvent,
    edge: DependencyFlowEdge,
    handleType: HandleType,
  ) => void;
  onReconnectEnd: (
    event: MouseEvent | TouchEvent,
    edge: DependencyFlowEdge,
    handleType: HandleType,
    connectionState: FinalConnectionState,
  ) => void;
  onMoveEnd: OnMove;
  connectionPending: boolean;
  layoutPending: boolean;
  connectionError: Error | null;
  taskTitle: (taskId: string) => string | undefined;
  layoutError: { message: string | undefined } | undefined;
  retryLayout: () => void;
  onCreateTask: () => void;
  ariaLabelConfig: Partial<AriaLabelConfig>;
  readOnly: boolean;
}

function GraphCanvas({
  tasks,
  hiddenTaskCount,
  nodes,
  edges,
  initialGraphZoom,
  onInit,
  onConnect,
  onConnectStart,
  onConnectEnd,
  onEdgesChange,
  onEdgesDelete,
  onEdgeClick,
  onNodeClick,
  onPaneClick,
  onReconnect,
  onReconnectStart,
  onReconnectEnd,
  onMoveEnd,
  connectionPending,
  layoutPending,
  connectionError,
  taskTitle,
  layoutError,
  retryLayout,
  onCreateTask,
  ariaLabelConfig,
  readOnly,
}: GraphCanvasProps) {
  const graphBusy = connectionPending || layoutPending;
  return (
    <div
      className="graph-stage"
      data-testid="feature-graph"
      aria-busy={graphBusy}
    >
      <ReactFlow<TaskFlowNode, DependencyFlowEdge>
        ariaLabelConfig={ariaLabelConfig}
        autoPanOnConnect={false}
        defaultEdgeOptions={{ animated: false }}
        defaultViewport={{ x: 0, y: 0, zoom: initialGraphZoom }}
        deleteKeyCode={["Backspace", "Delete"]}
        edges={edges}
        edgeTypes={edgeTypes}
        elevateEdgesOnSelect
        maxZoom={maxGraphZoom}
        minZoom={minGraphZoom}
        nodes={nodes}
        nodesConnectable={!readOnly && !graphBusy}
        nodesDraggable={false}
        nodeTypes={nodeTypes}
        onConnect={onConnect}
        onConnectStart={onConnectStart}
        onConnectEnd={onConnectEnd}
        onEdgesChange={onEdgesChange}
        onEdgesDelete={onEdgesDelete}
        onEdgeClick={onEdgeClick}
        onInit={onInit}
        onMoveEnd={onMoveEnd}
        onNodeClick={onNodeClick}
        onPaneClick={onPaneClick}
        onReconnect={onReconnect}
        onReconnectEnd={onReconnectEnd}
        onReconnectStart={onReconnectStart}
        proOptions={{ hideAttribution: true }}
        reconnectRadius={12}
      >
        <Background
          variant={BackgroundVariant.Dots}
          gap={24}
          size={1}
          color="var(--border)"
        />
        <Controls showInteractive={false} />
      </ReactFlow>
      {connectionError && (
        <div className="graph-connection-error">
          <MutationError error={connectionError} taskTitle={taskTitle} />
        </div>
      )}
      <GraphState
        taskCount={tasks.length}
        hiddenTaskCount={hiddenTaskCount}
        layoutError={layoutError}
        onCreateTask={onCreateTask}
        onRetryLayout={retryLayout}
        readOnly={readOnly}
      />
    </div>
  );
}

function GraphState({
  taskCount,
  hiddenTaskCount,
  layoutError,
  onCreateTask,
  onRetryLayout,
  readOnly,
}: {
  taskCount: number;
  hiddenTaskCount: number;
  layoutError: { message: string | undefined } | undefined;
  onCreateTask: () => void;
  onRetryLayout: () => void;
  readOnly: boolean;
}) {
  const { t } = useTranslation();
  // 全タスクが完了した feature にもグラフはあり、非表示は読み手が選んだ見え方
  // にすぎない。ここで最初のノードを促すと feature が空だと言うことになり、
  // 本来はトグルで解決する場面にタスク追加ボタンを置いてしまう。
  if (taskCount === 0 && hiddenTaskCount > 0 && !layoutError)
    return (
      <div className="graph-empty">
        <span>
          <EyeOff aria-hidden="true" focusable="false" size={24} />
        </span>
        <h2>{t("workspace.graphAllHiddenTitle")}</h2>
        <p>{t("workspace.graphAllHiddenDetail", { total: hiddenTaskCount })}</p>
      </div>
    );
  if (taskCount === 0 && !layoutError)
    return (
      <div className="graph-empty">
        <span>
          <Plus aria-hidden="true" focusable="false" size={24} />
        </span>
        <h2>{t("workspace.graphEmptyTitle")}</h2>
        <p>{t("workspace.graphEmptyDetail")}</p>
        {!readOnly && (
          <IconButton
            icon={Plus}
            label={t("workspace.addTaskPlain")}
            variant="primary"
            onClick={() => {
              onCreateTask();
            }}
          />
        )}
      </div>
    );
  if (layoutError)
    return (
      <div className="graph-empty" role="alert">
        <span>
          <TriangleAlert aria-hidden="true" focusable="false" size={24} />
        </span>
        <h2>{t("workspace.layoutErrorTitle")}</h2>
        <p>{layoutError.message ?? t("workspace.layoutErrorFallback")}</p>
        <IconButton
          icon={RotateCcw}
          label={t("workspace.retryLayout")}
          variant="secondary"
          onClick={onRetryLayout}
        />
      </div>
    );
  return null;
}

function GraphStatusNotice({
  addingDependency,
  connecting,
  detachingDependency,
  removingDependency,
  readOnly,
  taskTitle,
}: {
  addingDependency: boolean;
  connecting: boolean;
  detachingDependency: { blocker: string; blocked: string } | undefined;
  removingDependency: boolean;
  readOnly: boolean;
  taskTitle: (taskId: string) => string | undefined;
}) {
  const { t } = useTranslation();
  const instruction = removingDependency
    ? t("workspace.flow.dependencyRemoving")
    : addingDependency
      ? t("workspace.flow.connectionSaving")
      : detachingDependency
        ? t("workspace.flow.detachInstruction", {
            blocker:
              taskTitle(detachingDependency.blocker) ??
              detachingDependency.blocker,
            blocked:
              taskTitle(detachingDependency.blocked) ??
              detachingDependency.blocked,
          })
        : connecting
          ? t("workspace.flow.connectInstruction")
          : undefined;
  if (readOnly || !instruction) return null;
  return (
    <div className="graph-status-notice">
      <p
        className={`graph-connection-help ${addingDependency || removingDependency ? "is-saving" : ""}`}
        aria-live="polite"
        title={instruction}
      >
        {instruction}
      </p>
    </div>
  );
}
