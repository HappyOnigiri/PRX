import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import {
  MarkerType,
  type AriaLabelConfig,
  type Edge,
  type OnMoveEnd,
  type ReactFlowInstance,
} from "@xyflow/react";
import { useTranslation } from "react-i18next";
import {
  maxGraphZoom,
  minGraphZoom,
  readGraphZoom,
  writeGraphZoom,
} from "../i18n/settings";
import type { Dependency } from "../gen/prx/v1/prx_pb";
import type { TaskFlowNode } from "./TaskNode";

interface FeatureGraphControlsOptions {
  dependencies: Dependency[];
  nodes: TaskFlowNode[];
}

interface FeatureGraphControls {
  edges: Edge[];
  initialGraphZoom: number;
  flow: {
    set: (instance: ReactFlowInstance<TaskFlowNode>) => void;
    minZoom: number;
    maxZoom: number;
  };
  onMoveEnd: OnMoveEnd;
  ariaLabelConfig: Partial<AriaLabelConfig>;
}

export function useFeatureGraphControls({
  dependencies,
  nodes,
}: FeatureGraphControlsOptions): FeatureGraphControls {
  const { t } = useTranslation();
  const [flowInstance, setFlowInstance] =
    useState<ReactFlowInstance<TaskFlowNode>>();
  const [initialGraphZoom] = useState(readGraphZoom);
  const graphZoom = useRef(initialGraphZoom);
  const edges = useMemo(
    () =>
      dependencies.map((dependency) => ({
        id: `${dependency.blockerTaskId}-${dependency.blockedTaskId}`,
        source: dependency.blockerTaskId,
        target: dependency.blockedTaskId,
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
  const onMoveEnd = useCallback<OnMoveEnd>((_, viewport) => {
    graphZoom.current = viewport.zoom;
    writeGraphZoom(viewport.zoom);
  }, []);

  useEffect(() => {
    if (!flowInstance || nodes.length === 0) return;
    const bounds = flowInstance.getNodesBounds(nodes);
    const duration = window.matchMedia("(prefers-reduced-motion: reduce)")
      .matches
      ? 0
      : 260;
    void flowInstance.setCenter(
      bounds.x + bounds.width / 2,
      bounds.y + bounds.height / 2,
      { zoom: graphZoom.current, duration },
    );
  }, [flowInstance, nodes]);

  return {
    edges,
    initialGraphZoom,
    flow: {
      set: setFlowInstance,
      minZoom: minGraphZoom,
      maxZoom: maxGraphZoom,
    },
    onMoveEnd,
    ariaLabelConfig,
  };
}
