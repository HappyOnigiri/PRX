import ELK from "elkjs/lib/elk-api.js";
import elkWorkerUrl from "elkjs/lib/elk-worker.min.js?url";
import { useEffect, useState } from "react";
import type { Dependency, PullRequest, Task } from "../gen/prx/v1/prx_pb";
import { type TaskFlowNode, type TaskNodeDocument } from "./TaskNode";

interface GraphLayoutOptions {
  tasks: Task[];
  dependencies: Dependency[];
  pullRequests: Map<string, PullRequest>;
  documentsByTask: Map<string, TaskNodeDocument[]>;
  onEditTask: (taskId: string) => void;
  onPreviewDocument: (document: TaskNodeDocument) => void;
}

export interface GraphLayoutError {
  message: string | undefined;
}

interface RawTaskNode {
  id: string;
  width: number;
  height: number;
  data: TaskFlowNode["data"];
}

function createRawNodes(options: GraphLayoutOptions): RawTaskNode[] {
  return options.tasks.map((task) => {
    const pullRequest = options.pullRequests.get(task.id);
    const documents = options.documentsByTask.get(task.id) ?? [];
    const assetCount = documents.length + (pullRequest ? 1 : 0);
    return {
      id: task.id,
      width: 284,
      height: 148 + Math.min(assetCount, 4) * 34,
      data: {
        title: task.title,
        assignee: task.assignee,
        state: task.displayState,
        ready: task.ready,
        stale: pullRequest?.stale ?? false,
        pullRequest: pullRequest
          ? {
              label: `${pullRequest.owner}/${pullRequest.repository} #${String(pullRequest.number)}`,
              url: pullRequest.url,
            }
          : undefined,
        documents,
        onEdit: () => {
          options.onEditTask(task.id);
        },
        onPreview: options.onPreviewDocument,
      },
    };
  });
}

function createLayoutRequest(
  rawNodes: RawTaskNode[],
  dependencies: Dependency[],
) {
  const elk = new ELK({ workerUrl: elkWorkerUrl });
  const request = elk.layout({
    id: "root",
    layoutOptions: {
      "elk.algorithm": "layered",
      "elk.direction": "RIGHT",
      "elk.spacing.nodeNode": "72",
      "elk.layered.spacing.nodeNodeBetweenLayers": "110",
    },
    children: rawNodes.map(({ id, width, height }) => ({ id, width, height })),
    edges: dependencies.map((dependency, index) => ({
      id: String(index),
      sources: [dependency.blockerTaskId],
      targets: [dependency.blockedTaskId],
    })),
  });
  return {
    request,
    terminate: () => {
      elk.terminateWorker();
    },
  };
}

function positionNodes(
  rawNodes: RawTaskNode[],
  positions: Map<string, { x: number; y: number }>,
): TaskFlowNode[] {
  return rawNodes.map((node) => ({
    ...node,
    type: "task",
    position: positions.get(node.id) ?? { x: 0, y: 0 },
  }));
}

export function useGraphLayout({
  tasks,
  dependencies,
  pullRequests,
  documentsByTask,
  onEditTask,
  onPreviewDocument,
}: GraphLayoutOptions) {
  const [nodes, setNodes] = useState<TaskFlowNode[]>([]);
  // Keep the raw error so changing the display language does not re-run the
  // layout effect and reset the viewport.
  const [layoutError, setLayoutError] = useState<GraphLayoutError>();
  const [layoutAttempt, setLayoutAttempt] = useState(0);

  useEffect(() => {
    let current = true;
    const rawNodes = createRawNodes({
      tasks,
      dependencies,
      pullRequests,
      documentsByTask,
      onEditTask,
      onPreviewDocument,
    });
    const layout = createLayoutRequest(rawNodes, dependencies);
    layout.request
      .then((result) => {
        if (!current) return;
        setLayoutError(undefined);
        const positions = new Map(
          result.children?.map((child) => [
            child.id,
            { x: child.x ?? 0, y: child.y ?? 0 },
          ]),
        );
        setNodes(positionNodes(rawNodes, positions));
      })
      .catch((error: unknown) => {
        if (!current) return;
        setLayoutError({
          message: error instanceof Error ? error.message : undefined,
        });
      });
    return () => {
      current = false;
      layout.terminate();
    };
  }, [
    tasks,
    dependencies,
    pullRequests,
    documentsByTask,
    onEditTask,
    onPreviewDocument,
    layoutAttempt,
  ]);

  function retryLayout() {
    setLayoutError(undefined);
    setLayoutAttempt((attempt) => attempt + 1);
  }

  return { nodes, layoutError, retryLayout };
}
