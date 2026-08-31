import type { ElkNode } from "elkjs/lib/elk-api.js";
import ELK from "elkjs/lib/elk-api.js";
import elkWorkerUrl from "elkjs/lib/elk-worker.min.js?url";
import { useEffect, useMemo, useState } from "react";
import type { Dependency, PullRequest, Task } from "../gen/prx/v1/prx_pb";
import { dependencyEdgeId, type DependencyEdgeRoute } from "./dependencyGraph";
import { type TaskFlowNode, type TaskNodeDocument } from "./TaskNode";

interface GraphLayoutOptions {
  tasks: Task[];
  dependencies: Dependency[];
  pullRequests: Map<string, PullRequest>;
  documentsByTask: Map<string, TaskNodeDocument[]>;
  onEditTask: (taskId: string) => void;
  onPreviewDocument: (document: TaskNodeDocument) => void;
  onAddDocument?: (taskId: string, trigger: HTMLButtonElement) => void;
  readOnly?: boolean;
}

interface LayoutRequest extends GraphLayoutOptions {
  attempt: number;
}

function isSameLayoutRequest(
  completed: LayoutRequest | undefined,
  requested: LayoutRequest,
) {
  return (
    completed?.tasks === requested.tasks &&
    completed.dependencies === requested.dependencies &&
    completed.pullRequests === requested.pullRequests &&
    completed.documentsByTask === requested.documentsByTask &&
    completed.onEditTask === requested.onEditTask &&
    completed.onPreviewDocument === requested.onPreviewDocument &&
    completed.onAddDocument === requested.onAddDocument &&
    completed.readOnly === requested.readOnly &&
    completed.attempt === requested.attempt
  );
}

function buildRawNodes({
  tasks,
  pullRequests,
  documentsByTask,
  onEditTask,
  onPreviewDocument,
  onAddDocument,
  readOnly = false,
}: GraphLayoutOptions) {
  return tasks.map((task) => {
    const pr = pullRequests.get(task.id);
    const documents = documentsByTask.get(task.id) ?? [];
    const assetCount = documents.length + (pr ? 1 : 0) + (readOnly ? 0 : 1);
    const hasSyncError = Boolean(pr?.syncError);
    return {
      id: task.id,
      width: 284,
      height: 148 + Math.min(assetCount, 4) * 34 + (hasSyncError ? 22 : 0),
      data: {
        title: task.title,
        assignee: task.assignee,
        state: task.displayState,
        ready: task.ready,
        stale: pr?.stale ?? false,
        syncError: hasSyncError,
        pullRequest: pr
          ? {
              label: `${pr.host && pr.host !== "github.com" ? `${pr.host}/` : ""}${pr.owner}/${pr.repository} #${String(pr.number)}`,
              url: pr.url,
            }
          : undefined,
        documents,
        readOnly,
        onEdit: () => {
          onEditTask(task.id);
        },
        onPreview: onPreviewDocument,
        onAddReference: (trigger: HTMLButtonElement) => {
          onAddDocument?.(task.id, trigger);
        },
      },
    };
  });
}

type RawNode = ReturnType<typeof buildRawNodes>[number];

function buildLayoutGraph(raw: RawNode[], dependencies: Dependency[]): ElkNode {
  return {
    id: "root",
    layoutOptions: {
      "elk.algorithm": "layered",
      "elk.direction": "RIGHT",
      "elk.edgeRouting": "ORTHOGONAL",
      "elk.layered.mergeEdges": "false",
      "elk.spacing.componentComponent": "120",
      "elk.spacing.nodeNode": "72",
      "elk.layered.spacing.nodeNodeBetweenLayers": "110",
    },
    children: raw.map(({ id, width, height }) => ({ id, width, height })),
    edges: dependencies.map((dependency) => ({
      id: dependencyEdgeId(dependency.blockerTaskId, dependency.blockedTaskId),
      sources: [dependency.blockerTaskId],
      targets: [dependency.blockedTaskId],
    })),
  };
}

function readLayout(
  layout: ElkNode,
  raw: RawNode[],
  dependencies: Dependency[],
) {
  const positions = new Map(
    layout.children?.map((child) => [
      child.id,
      { x: child.x ?? 0, y: child.y ?? 0 },
    ]),
  );
  const ports = new Map<
    string,
    {
      incoming: { id: string; top: number }[];
      outgoing: { id: string; top: number }[];
    }
  >(raw.map((node) => [node.id, { incoming: [], outgoing: [] }]));
  const edgeRoutes = new Map<string, DependencyEdgeRoute>();

  for (const dependency of dependencies) {
    const id = dependencyEdgeId(
      dependency.blockerTaskId,
      dependency.blockedTaskId,
    );
    const section = layout.edges?.find((edge) => edge.id === id)?.sections?.[0];
    const sourcePosition = positions.get(dependency.blockerTaskId);
    const targetPosition = positions.get(dependency.blockedTaskId);
    if (!section || !sourcePosition || !targetPosition) continue;

    const sourcePortId = `${id}-source`;
    const targetPortId = `${id}-target`;
    const route = {
      points: [
        section.startPoint,
        ...(section.bendPoints ?? []),
        section.endPoint,
      ],
      sourcePortId,
      sourcePortTop: section.startPoint.y - sourcePosition.y,
      targetPortId,
      targetPortTop: section.endPoint.y - targetPosition.y,
    };
    edgeRoutes.set(id, route);
    ports.get(dependency.blockerTaskId)?.outgoing.push({
      id: sourcePortId,
      top: route.sourcePortTop,
    });
    ports.get(dependency.blockedTaskId)?.incoming.push({
      id: targetPortId,
      top: route.targetPortTop,
    });
  }

  const nodes: TaskFlowNode[] = raw.map((node) => ({
    ...node,
    data: {
      ...node.data,
      incomingPorts: ports.get(node.id)?.incoming ?? [],
      outgoingPorts: ports.get(node.id)?.outgoing ?? [],
    },
    type: "task",
    position: positions.get(node.id) ?? { x: 0, y: 0 },
  }));
  return { edgeRoutes, nodes };
}

export function useGraphLayout({
  tasks,
  dependencies,
  pullRequests,
  documentsByTask,
  onEditTask,
  onPreviewDocument,
  onAddDocument,
  readOnly = false,
}: GraphLayoutOptions) {
  const [nodes, setNodes] = useState<TaskFlowNode[]>([]);
  const [edgeRoutes, setEdgeRoutes] = useState<
    Map<string, DependencyEdgeRoute>
  >(() => new Map());
  // Keep the raw error so changing the display language does not re-run the
  // layout effect and reset the viewport.
  const [layoutError, setLayoutError] = useState<
    { message: string | undefined } | undefined
  >();
  const [layoutAttempt, setLayoutAttempt] = useState(0);
  const [completedLayout, setCompletedLayout] = useState<LayoutRequest>();
  const layoutRequest = useMemo(
    () => ({
      tasks,
      dependencies,
      pullRequests,
      documentsByTask,
      onEditTask,
      onPreviewDocument,
      ...(onAddDocument ? { onAddDocument } : {}),
      readOnly,
      attempt: layoutAttempt,
    }),
    [
      tasks,
      dependencies,
      pullRequests,
      documentsByTask,
      onEditTask,
      onPreviewDocument,
      onAddDocument,
      readOnly,
      layoutAttempt,
    ],
  );
  const layoutPending = !isSameLayoutRequest(completedLayout, layoutRequest);

  useEffect(() => {
    let current = true;
    const raw = buildRawNodes(layoutRequest);
    const elk = new ELK({ workerUrl: elkWorkerUrl });
    elk
      .layout(buildLayoutGraph(raw, layoutRequest.dependencies))
      .then((layout) => {
        if (!current) return;
        setLayoutError(undefined);
        const result = readLayout(layout, raw, layoutRequest.dependencies);
        setEdgeRoutes(result.edgeRoutes);
        setNodes(result.nodes);
        setCompletedLayout(layoutRequest);
      })
      .catch((error: unknown) => {
        if (!current) return;
        setEdgeRoutes(new Map());
        setLayoutError({
          message: error instanceof Error ? error.message : undefined,
        });
        setCompletedLayout(layoutRequest);
      });
    return () => {
      current = false;
      elk.terminateWorker();
    };
  }, [layoutRequest]);

  function retryLayout() {
    setLayoutError(undefined);
    setLayoutAttempt((attempt) => attempt + 1);
  }

  return { edgeRoutes, nodes, layoutError, layoutPending, retryLayout };
}
