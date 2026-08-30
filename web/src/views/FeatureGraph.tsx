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
import { Plus, RotateCcw, TriangleAlert } from "lucide-react";
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
import { DependencyEdge } from "./DependencyEdge";
import {
  dependencyEdgeId,
  type DependencyEdgeRoute,
  type DependencyFlowEdge,
} from "./dependencyGraph";
import { IconButton } from "./IconButton";
import { MutationError } from "./MutationError";
import { TaskNode, type TaskFlowNode, type TaskNodeDocument } from "./TaskNode";
import { useGraphLayout } from "./useGraphLayout";

const nodeTypes = { task: TaskNode };
const edgeTypes = { dependency: DependencyEdge };

function useDependencyConnections(readOnly: boolean) {
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
  const pending = addDependency.isPending || removeDependency.isPending;
  const onConnect = useCallback(
    ({ source, target }: Connection) => {
      if (readOnly || pending || !source || !target) return;
      addDependency.mutate({ blocker: source, blocked: target });
    },
    [addDependency, pending, readOnly],
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
      setDetaching(undefined);
      if (connectionState.isValid === true) return;
      remove(edge);
    },
    [remove],
  );

  return {
    adding: addDependency.isPending,
    detaching,
    error: removeDependency.error ?? addDependency.error,
    onConnect,
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

// React Flow compares the edge array by reference, so a fresh array on every
// render rebuilds its connection lookup and re-renders every store consumer.
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

// React Flow measures the endpoint ports one animation frame after the nodes
// that carry them commit. Handing the routes to the edges only after that frame
// keeps the edges from naming handles that do not exist yet, which would drop
// them from the canvas until the measurement lands.
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
  // Dropping the id keeps a re-added dependency from reappearing as selected
  // without the user ever picking it.
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
  // React Flow drops its own selection updates because the edges are
  // controlled, so Enter and Space on a focused edge only reach the toolbar and
  // the Delete key once the change is applied here.
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
  onEditTask: (taskId: string) => void;
  onPreviewDocument: (document: TaskNodeDocument) => void;
  onCreateTask: () => void;
  readOnly?: boolean;
}

export function FeatureGraph({
  tasks,
  dependencies,
  pullRequests,
  documentsByTask,
  onEditTask,
  onPreviewDocument,
  onCreateTask,
  readOnly = false,
}: FeatureGraphProps) {
  const { t } = useTranslation();
  const [flow, setFlow] =
    useState<ReactFlowInstance<TaskFlowNode, DependencyFlowEdge>>();
  const [initialGraphZoom] = useState(readGraphZoom);
  const graphZoom = useRef(initialGraphZoom);
  const connections = useDependencyConnections(readOnly);
  const selection = useDependencySelection(dependencies);
  const taskTitle = useTaskTitle(tasks);
  const { edgeRoutes, nodes, layoutError, layoutPending, retryLayout } =
    useGraphLayout({
      tasks,
      dependencies,
      pullRequests,
      documentsByTask,
      onEditTask,
      onPreviewDocument,
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
      <GraphStatusBar
        taskCount={tasks.length}
        dependencyCount={dependencies.length}
        addingDependency={connections.adding}
        detachingDependency={connections.detaching}
        removingDependency={connections.removing}
        readOnly={readOnly}
        taskTitle={taskTitle}
      />
      <GraphCanvas
        tasks={tasks}
        nodes={nodes}
        edges={edges}
        initialGraphZoom={initialGraphZoom}
        onInit={setFlow}
        onConnect={connections.onConnect}
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
  nodes: TaskFlowNode[];
  edges: DependencyFlowEdge[];
  initialGraphZoom: number;
  onInit: (
    instance: ReactFlowInstance<TaskFlowNode, DependencyFlowEdge>,
  ) => void;
  onConnect: (connection: Connection) => void;
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
  nodes,
  edges,
  initialGraphZoom,
  onInit,
  onConnect,
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
  layoutError,
  onCreateTask,
  onRetryLayout,
  readOnly,
}: {
  taskCount: number;
  layoutError: { message: string | undefined } | undefined;
  onCreateTask: () => void;
  onRetryLayout: () => void;
  readOnly: boolean;
}) {
  const { t } = useTranslation();
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
            onClick={onCreateTask}
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

function GraphStatusBar({
  taskCount,
  dependencyCount,
  addingDependency,
  detachingDependency,
  removingDependency,
  readOnly,
  taskTitle,
}: {
  taskCount: number;
  dependencyCount: number;
  addingDependency: boolean;
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
        : t("workspace.flow.connectionInstruction");
  return (
    <div className="graph-status-bar">
      {!readOnly && (
        <p
          className={`graph-connection-help ${addingDependency || removingDependency ? "is-saving" : ""}`}
          aria-live="polite"
          title={instruction}
        >
          {instruction}
        </p>
      )}
      <b>
        {t("workspace.graphSummary", {
          nodes: taskCount,
          links: dependencyCount,
        })}
      </b>
    </div>
  );
}
