import {
  Background,
  BackgroundVariant,
  Controls,
  ReactFlow,
} from "@xyflow/react";
import type { Dependency, PullRequest, Task } from "../gen/prx/v1/prx_pb";
import { FeatureGraphLegend } from "./FeatureGraphLegend";
import { FeatureGraphState } from "./FeatureGraphState";
import { TaskNode, type TaskFlowNode, type TaskNodeDocument } from "./TaskNode";
import { useFeatureGraphControls } from "./useFeatureGraphControls";
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
  const { nodes, layoutError, retryLayout } = useGraphLayout({
    tasks,
    dependencies,
    pullRequests,
    documentsByTask,
    onEditTask,
    onPreviewDocument,
  });
  const { edges, initialGraphZoom, flow, onMoveEnd, ariaLabelConfig } =
    useFeatureGraphControls({ dependencies, nodes });

  return (
    <>
      <FeatureGraphLegend
        taskCount={tasks.length}
        dependencyCount={dependencies.length}
      />
      <div className="graph-stage" data-testid="feature-graph">
        <ReactFlow<TaskFlowNode>
          nodes={nodes}
          edges={edges}
          nodeTypes={nodeTypes}
          onInit={flow.set}
          defaultViewport={{ x: 0, y: 0, zoom: initialGraphZoom }}
          onMoveEnd={onMoveEnd}
          minZoom={flow.minZoom}
          maxZoom={flow.maxZoom}
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
        <FeatureGraphState
          taskCount={tasks.length}
          layoutError={layoutError}
          onCreateTask={onCreateTask}
          onRetryLayout={retryLayout}
        />
      </div>
    </>
  );
}
