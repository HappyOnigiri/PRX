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
  const [layoutError, setLayoutError] = useState<
    { message: string | undefined } | undefined
  >();
  const [layoutAttempt, setLayoutAttempt] = useState(0);

  useEffect(() => {
    let current = true;
    const raw = tasks.map((task) => {
      const pr = pullRequests.get(task.id);
      const documents = documentsByTask.get(task.id) ?? [];
      const assetCount = documents.length + (pr ? 1 : 0);
      return {
        id: task.id,
        width: 284,
        height: 148 + Math.min(assetCount, 4) * 34,
        data: {
          title: task.title,
          assignee: task.assignee,
          state: task.displayState,
          ready: task.ready,
          stale: pr?.stale ?? false,
          pullRequest: pr
            ? {
                label: `${pr.owner}/${pr.repository} #${String(pr.number)}`,
                url: pr.url,
              }
            : undefined,
          documents,
          onEdit: () => {
            onEditTask(task.id);
          },
          onPreview: onPreviewDocument,
        },
      };
    });
    const elk = new ELK({ workerUrl: elkWorkerUrl });
    elk
      .layout({
        id: "root",
        layoutOptions: {
          "elk.algorithm": "layered",
          "elk.direction": "RIGHT",
          "elk.spacing.nodeNode": "72",
          "elk.layered.spacing.nodeNodeBetweenLayers": "110",
        },
        children: raw.map(({ id, width, height }) => ({ id, width, height })),
        edges: dependencies.map((dep, index) => ({
          id: String(index),
          sources: [dep.blockerTaskId],
          targets: [dep.blockedTaskId],
        })),
      })
      .then((layout) => {
        if (!current) return;
        setLayoutError(undefined);
        const positions = new Map(
          layout.children?.map((child) => [
            child.id,
            { x: child.x ?? 0, y: child.y ?? 0 },
          ]),
        );
        setNodes(
          raw.map((node) => ({
            ...node,
            type: "task",
            position: positions.get(node.id) ?? { x: 0, y: 0 },
          })),
        );
      })
      .catch((error: unknown) => {
        if (!current) return;
        setLayoutError({
          message: error instanceof Error ? error.message : undefined,
        });
      });
    return () => {
      current = false;
      elk.terminateWorker();
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
