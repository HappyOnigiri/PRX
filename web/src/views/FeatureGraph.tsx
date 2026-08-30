import {
  Background,
  BackgroundVariant,
  Controls,
  MarkerType,
  ReactFlow,
  type Edge,
  type ReactFlowInstance,
} from "@xyflow/react";
import { Plus, RotateCcw, TriangleAlert } from "lucide-react";
import { useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import type { Dependency, PullRequest, Task } from "../gen/prx/v1/prx_pb";
import {
  maxGraphZoom,
  minGraphZoom,
  readGraphZoom,
  writeGraphZoom,
} from "../i18n/settings";
import { IconButton } from "./IconButton";
import { TaskNode, type TaskFlowNode, type TaskNodeDocument } from "./TaskNode";
import { useGraphLayout } from "./useGraphLayout";

const nodeTypes = { task: TaskNode };

interface FeatureGraphProps {
  tasks: Task[];
  dependencies: Dependency[];
  pullRequests: Map<string, PullRequest>;
  documentsByTask: Map<string, TaskNodeDocument[]>;
  onEditTask: (taskId: string) => void;
  onPreviewDocument: (document: TaskNodeDocument) => void;
  onCreateTask: () => void;
}

export function FeatureGraph({
  tasks,
  dependencies,
  pullRequests,
  documentsByTask,
  onEditTask,
  onPreviewDocument,
  onCreateTask,
}: FeatureGraphProps) {
  const { t } = useTranslation();
  const [flow, setFlow] = useState<ReactFlowInstance<TaskFlowNode>>();
  const [initialGraphZoom] = useState(readGraphZoom);
  const graphZoom = useRef(initialGraphZoom);
  const { nodes, layoutError, retryLayout } = useGraphLayout({
    tasks,
    dependencies,
    pullRequests,
    documentsByTask,
    onEditTask,
    onPreviewDocument,
  });
  const edges: Edge[] = useMemo(
    () =>
      dependencies.map((dep) => ({
        id: `${dep.blockerTaskId}-${dep.blockedTaskId}`,
        source: dep.blockerTaskId,
        target: dep.blockedTaskId,
        type: "smoothstep",
        markerEnd: { type: MarkerType.ArrowClosed },
        className: "dependency-edge",
      })),
    [dependencies],
  );
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
      <GraphLegend
        taskCount={tasks.length}
        dependencyCount={dependencies.length}
      />
      <div className="graph-stage" data-testid="feature-graph">
        <ReactFlow
          nodes={nodes}
          edges={edges}
          nodeTypes={nodeTypes}
          onInit={setFlow}
          defaultViewport={{ x: 0, y: 0, zoom: initialGraphZoom }}
          onMoveEnd={(_, viewport) => {
            graphZoom.current = viewport.zoom;
            writeGraphZoom(viewport.zoom);
          }}
          minZoom={minGraphZoom}
          maxZoom={maxGraphZoom}
          nodesDraggable={false}
          defaultEdgeOptions={{ animated: false }}
          ariaLabelConfig={ariaLabelConfig}
        >
          <Background
            variant={BackgroundVariant.Dots}
            gap={24}
            size={1}
            color="var(--border)"
          />
          <Controls showInteractive={false} />
        </ReactFlow>
        <GraphState
          taskCount={tasks.length}
          layoutError={layoutError}
          onCreateTask={onCreateTask}
          onRetryLayout={retryLayout}
        />
      </div>
    </>
  );
}

function GraphState({
  taskCount,
  layoutError,
  onCreateTask,
  onRetryLayout,
}: {
  taskCount: number;
  layoutError: { message: string | undefined } | undefined;
  onCreateTask: () => void;
  onRetryLayout: () => void;
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
        <IconButton
          icon={Plus}
          label={t("workspace.addTaskPlain")}
          variant="primary"
          onClick={onCreateTask}
        />
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

function GraphLegend({
  taskCount,
  dependencyCount,
}: {
  taskCount: number;
  dependencyCount: number;
}) {
  const { t } = useTranslation();
  return (
    <div className="graph-legend">
      <span>
        <i className="ready" />
        {t("workspace.legend.ready")}
      </span>
      <span>
        <i className="review" />
        {t("workspace.legend.review")}
      </span>
      <span>
        <i className="conflict" />
        {t("workspace.legend.conflict")}
      </span>
      <span>
        <i className="merged" />
        {t("workspace.legend.merged")}
      </span>
      <b>
        {t("workspace.graphSummary", {
          nodes: taskCount,
          links: dependencyCount,
        })}
      </b>
    </div>
  );
}
