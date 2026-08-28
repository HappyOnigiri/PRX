import { FormEvent, useEffect, useMemo, useState } from "react";
import { useNavigate, useParams } from "@tanstack/react-router";
import {
  Background,
  BackgroundVariant,
  Controls,
  MarkerType,
  ReactFlow,
  type Edge,
  type ReactFlowInstance,
} from "@xyflow/react";
import ELK from "elkjs/lib/elk-api.js";
import elkWorkerUrl from "elkjs/lib/elk-worker.min.js?url";
import { mutations } from "../api";
import { useDomainMutation, useSnapshot } from "../hooks";
import { TaskNode, type TaskFlowNode } from "./TaskNode";

const nodeTypes = { task: TaskNode };

export function FeatureWorkspace() {
  const { featureId } = useParams({ from: "/features/$featureId" });
  const navigate = useNavigate();
  const snapshot = useSnapshot();
  const [selected, setSelected] = useState<string>();
  const [nodes, setNodes] = useState<TaskFlowNode[]>([]);
  const [flow, setFlow] = useState<ReactFlowInstance<TaskFlowNode, Edge>>();
  const [showTask, setShowTask] = useState(false);
  const [showFeatureEdit, setShowFeatureEdit] = useState(false);
  const data = snapshot.data;
  const feature = data?.features.find((f) => f.id === featureId);
  const tasks = useMemo(
    () => data?.tasks.filter((task) => task.featureId === featureId) ?? [],
    [data, featureId],
  );
  const taskIds = useMemo(() => new Set(tasks.map((task) => task.id)), [tasks]);
  const dependencies = useMemo(
    () =>
      data?.dependencies.filter((dep) => taskIds.has(dep.blockerTaskId)) ?? [],
    [data, taskIds],
  );
  const prs = useMemo(
    () => new Map(data?.pullRequests.map((pr) => [pr.taskId, pr]) ?? []),
    [data],
  );
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
  useEffect(() => {
    let current = true;
    const raw = tasks.map((task) => {
      const pr = prs.get(task.id);
      return {
        id: task.id,
        width: 244,
        height: 126,
        data: {
          title: task.title,
          repository: pr ? `${pr.owner}/${pr.repository} #${pr.number}` : "",
          assignee: task.assignee,
          state: task.displayState,
          ready: task.ready,
          stale: pr?.stale ?? false,
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
      });
    return () => {
      current = false;
      elk.terminateWorker();
    };
  }, [tasks, dependencies, prs]);
  useEffect(() => {
    if (nodes.length && flow) {
      void flow.fitView({
        padding: 0.16,
        duration: window.matchMedia("(prefers-reduced-motion: reduce)").matches
          ? 0
          : 260,
      });
    }
  }, [nodes, flow]);
  const createTask = useDomainMutation(mutations.createTask);
  const sync = useDomainMutation((id: string) => mutations.sync(id));
  const updateFeature = useDomainMutation(mutations.updateFeature);
  const deleteFeature = useDomainMutation(mutations.deleteFeature);
  if (snapshot.isPending)
    return (
      <div className="state-message">
        <div className="spinner" />
        <h1>Tracing feature graph…</h1>
      </div>
    );
  if (!feature || !data)
    return (
      <div className="state-message">
        <h1>Feature not found</h1>
        <button onClick={() => navigate({ to: "/" })}>
          Return to overview
        </button>
      </div>
    );
  async function submitTask(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = new FormData(event.currentTarget);
    await createTask.mutateAsync({
      featureId,
      title: String(form.get("title")),
      scope: String(form.get("scope")),
      kind: String(form.get("kind")),
      assignee: String(form.get("assignee")),
    });
    setShowTask(false);
  }
  async function submitFeature(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = new FormData(event.currentTarget);
    await updateFeature.mutateAsync({
      id: featureId,
      slug: String(form.get("slug")),
      title: String(form.get("title")),
      description: String(form.get("description")),
      status: String(form.get("status")),
    });
    setShowFeatureEdit(false);
  }
  async function removeFeature() {
    if (!window.confirm(`Delete ${feature?.title} and every contained task?`))
      return;
    await deleteFeature.mutateAsync(featureId);
    await navigate({ to: "/" });
  }
  const selectedTask = tasks.find((task) => task.id === selected);
  return (
    <div className="workspace">
      <header className="workspace-head">
        <div>
          <p className="eyebrow">Feature circuit / {feature.slug}</p>
          <h1>{feature.title}</h1>
          <p>{feature.description || "No feature description yet."}</p>
        </div>
        <div className="workspace-actions">
          <button
            className="secondary"
            onClick={() => sync.mutate(featureId)}
            disabled={sync.isPending}
          >
            {sync.isPending ? "Syncing…" : "↻ Sync GitHub"}
          </button>
          <button onClick={() => setShowTask(true)}>＋ Add task</button>
          <button
            className="icon-button"
            aria-label="Edit feature"
            onClick={() => setShowFeatureEdit(true)}
          >
            ✎
          </button>
          <button
            className="icon-button"
            aria-label={
              feature.archived ? "Unarchive feature" : "Archive feature"
            }
            onClick={() =>
              updateFeature.mutate({
                id: featureId,
                archived: !feature.archived,
              })
            }
          >
            ⌁
          </button>
          <button
            className="icon-button danger"
            aria-label="Delete feature"
            onClick={removeFeature}
          >
            ×
          </button>
        </div>
      </header>
      <div className="graph-legend">
        <span>
          <i className="ready" />
          Ready
        </span>
        <span>
          <i className="review" />
          Review
        </span>
        <span>
          <i className="conflict" />
          Conflict
        </span>
        <span>
          <i className="merged" />
          Merged
        </span>
        <b>
          {tasks.length} nodes · {dependencies.length} links
        </b>
      </div>
      <div className="graph-stage" data-testid="feature-graph">
        <ReactFlow
          nodes={nodes}
          edges={edges}
          nodeTypes={nodeTypes}
          onInit={setFlow}
          onNodeClick={(_, node) => setSelected(node.id)}
          fitView
          minZoom={0.08}
          maxZoom={1.7}
          nodesDraggable={false}
          defaultEdgeOptions={{ animated: false }}
        >
          <Background variant={BackgroundVariant.Dots} gap={22} size={1} />
          <Controls showInteractive={false} />
        </ReactFlow>
        {tasks.length === 0 && (
          <div className="graph-empty">
            <span>＋</span>
            <h2>Draw the first node</h2>
            <p>Add an implementation PR or a manual gate.</p>
            <button onClick={() => setShowTask(true)}>Add task</button>
          </div>
        )}
      </div>
      {selectedTask && (
        <TaskInspector
          task={selectedTask}
          tasks={tasks}
          dependencies={dependencies}
          pr={prs.get(selectedTask.id)}
          documents={data.documents.filter(
            (document) => document.taskId === selectedTask.id,
          )}
          onClose={() => setSelected(undefined)}
        />
      )}
      {showTask && (
        <div className="scrim">
          <form
            className="dialog"
            onSubmit={submitTask}
            aria-label="Create task"
          >
            <header>
              <p>New graph node</p>
              <h2>Add task</h2>
            </header>
            <label>
              Title
              <input
                name="title"
                required
                placeholder="Implement checkout API"
              />
            </label>
            <label>
              Scope
              <textarea
                name="scope"
                placeholder="Repository and acceptance boundary"
              />
            </label>
            <div className="form-row">
              <label>
                Kind
                <select name="kind">
                  <option value="pr">Pull request</option>
                  <option value="manual">Manual gate</option>
                </select>
              </label>
              <label>
                Assignee
                <input name="assignee" placeholder="Mika" />
              </label>
            </div>
            {createTask.error && (
              <p className="form-error">{createTask.error.message}</p>
            )}
            <footer>
              <button
                type="button"
                className="secondary"
                onClick={() => setShowTask(false)}
              >
                Cancel
              </button>
              <button disabled={createTask.isPending}>Add task</button>
            </footer>
          </form>
        </div>
      )}
      {showFeatureEdit && (
        <div className="scrim">
          <form
            className="dialog"
            onSubmit={submitFeature}
            aria-label="Edit feature"
          >
            <header>
              <p>Feature settings</p>
              <h2>Edit feature</h2>
            </header>
            <label>
              Slug
              <input name="slug" required defaultValue={feature.slug} />
            </label>
            <label>
              Title
              <input name="title" required defaultValue={feature.title} />
            </label>
            <label>
              Description
              <textarea name="description" defaultValue={feature.description} />
            </label>
            <label>
              Status
              <select name="status" defaultValue={feature.status}>
                <option value="active">Active</option>
                <option value="paused">Paused</option>
                <option value="completed">Completed</option>
                <option value="cancelled">Cancelled</option>
              </select>
            </label>
            {updateFeature.error && (
              <p className="form-error">{updateFeature.error.message}</p>
            )}
            <footer>
              <button
                type="button"
                className="secondary"
                onClick={() => setShowFeatureEdit(false)}
              >
                Cancel
              </button>
              <button disabled={updateFeature.isPending}>Save feature</button>
            </footer>
          </form>
        </div>
      )}
    </div>
  );
}

type InspectorProps = {
  task: {
    id: string;
    title: string;
    scope: string;
    status: string;
    assignee: string;
    displayState: string;
    blockedReason: string;
  };
  tasks: Array<{ id: string; title: string }>;
  dependencies: Array<{ blockerTaskId: string; blockedTaskId: string }>;
  pr?: {
    url: string;
    owner: string;
    repository: string;
    number: bigint;
    displayState: string;
    syncError: string;
    stale: boolean;
  };
  documents: Array<{ id: string; kind: string; title: string; value: string }>;
  onClose: () => void;
};
function TaskInspector({
  task,
  tasks,
  dependencies,
  pr,
  documents,
  onClose,
}: InspectorProps) {
  const update = useDomainMutation(mutations.updateTask);
  const remove = useDomainMutation(mutations.deleteTask);
  const attach = useDomainMutation(
    ({ taskId, url }: { taskId: string; url: string }) =>
      mutations.attachPR(taskId, url),
  );
  const detach = useDomainMutation(mutations.detachPR);
  const addDep = useDomainMutation(
    ({ blocker, blocked }: { blocker: string; blocked: string }) =>
      mutations.addDependency(blocker, blocked),
  );
  const removeDep = useDomainMutation(
    ({ blocker, blocked }: { blocker: string; blocked: string }) =>
      mutations.removeDependency(blocker, blocked),
  );
  const addDoc = useDomainMutation(mutations.addDocument);
  const deleteDoc = useDomainMutation(mutations.deleteDocument);
  const blockers = dependencies.filter((d) => d.blockedTaskId === task.id);
  return (
    <aside className="inspector" aria-label="Task inspector">
      <header>
        <div>
          <p>Node inspector</p>
          <h2>{task.title}</h2>
        </div>
        <button aria-label="Close inspector" onClick={onClose}>
          ×
        </button>
      </header>
      <div className={`inspector-state state-${task.displayState}`}>
        <i />
        {task.displayState.replaceAll("_", " ")}
        {task.blockedReason && <small>{task.blockedReason}</small>}
      </div>
      <form
        onSubmit={(e) => {
          e.preventDefault();
          const f = new FormData(e.currentTarget);
          update.mutate({
            id: task.id,
            title: String(f.get("title")),
            scope: String(f.get("scope")),
            status: String(f.get("status")),
            assignee: String(f.get("assignee")),
          });
        }}
      >
        <label>
          Title
          <input name="title" defaultValue={task.title} />
        </label>
        <label>
          Scope
          <textarea name="scope" defaultValue={task.scope} />
        </label>
        <div className="form-row">
          <label>
            Status
            <select name="status" defaultValue={task.status}>
              <option value="planned">Planned</option>
              <option value="in_progress">In progress</option>
              <option value="completed">Completed</option>
              <option value="cancelled">Cancelled</option>
            </select>
          </label>
          <label>
            Assignee
            <input name="assignee" defaultValue={task.assignee} />
          </label>
        </div>
        <button disabled={update.isPending}>Save task</button>
      </form>
      <section>
        <h3>Pull request</h3>
        {pr ? (
          <div className="linked-pr">
            <a href={pr.url} target="_blank" rel="noreferrer">
              {pr.owner}/{pr.repository} #{String(pr.number)}
            </a>
            <span>{pr.stale ? "stale" : pr.displayState}</span>
            {pr.syncError && <p>{pr.syncError}</p>}
            <button
              className="text-action"
              onClick={() => detach.mutate(task.id)}
            >
              Detach
            </button>
          </div>
        ) : (
          <form
            className="inline-form"
            onSubmit={(e) => {
              e.preventDefault();
              attach.mutate({
                taskId: task.id,
                url: String(new FormData(e.currentTarget).get("url")),
              });
            }}
          >
            <input
              name="url"
              required
              placeholder="https://github.com/org/repo/pull/42"
            />
            <button>Attach</button>
          </form>
        )}
      </section>
      <section>
        <h3>Blocked by</h3>
        {blockers.map((dep) => (
          <div className="dependency-chip" key={dep.blockerTaskId}>
            <span>{tasks.find((t) => t.id === dep.blockerTaskId)?.title}</span>
            <button
              aria-label="Remove dependency"
              onClick={() =>
                removeDep.mutate({
                  blocker: dep.blockerTaskId,
                  blocked: task.id,
                })
              }
            >
              ×
            </button>
          </div>
        ))}
        <form
          className="inline-form"
          onSubmit={(e) => {
            e.preventDefault();
            addDep.mutate({
              blocker: String(new FormData(e.currentTarget).get("blocker")),
              blocked: task.id,
            });
          }}
        >
          <select name="blocker" aria-label="Blocker task" defaultValue="">
            <option value="" disabled>
              Select blocker
            </option>
            {tasks
              .filter(
                (candidate) =>
                  candidate.id !== task.id &&
                  !blockers.some((d) => d.blockerTaskId === candidate.id),
              )
              .map((candidate) => (
                <option key={candidate.id} value={candidate.id}>
                  {candidate.title}
                </option>
              ))}
          </select>
          <button>Add</button>
        </form>
        {addDep.error && (
          <p className="form-error" role="alert">
            {addDep.error.message}
          </p>
        )}
      </section>
      <section>
        <h3>References</h3>
        {documents.map((document) => (
          <div className="document-chip" key={document.id}>
            <span>
              <b>{document.title || document.kind}</b>
              <small>{document.value}</small>
            </span>
            <button
              aria-label={`Delete ${document.title || "reference"}`}
              onClick={() => deleteDoc.mutate(document.id)}
            >
              ×
            </button>
          </div>
        ))}
        <form
          className="stack-form"
          onSubmit={(e) => {
            e.preventDefault();
            const f = new FormData(e.currentTarget);
            addDoc.mutate({
              taskId: task.id,
              kind: String(f.get("kind")),
              title: String(f.get("title")),
              value: String(f.get("value")),
            });
          }}
        >
          <div className="form-row">
            <select name="kind">
              <option value="url">URL</option>
              <option value="markdown_path">Markdown path</option>
            </select>
            <input name="title" placeholder="Design notes" />
          </div>
          <input
            name="value"
            required
            placeholder="https://… or docs/plan.md"
          />
          <button>Add reference</button>
        </form>
      </section>
      <button
        className="danger-zone"
        onClick={() => {
          if (window.confirm(`Delete ${task.title}?`)) {
            remove.mutate(task.id);
            onClose();
          }
        }}
      >
        Delete task and references
      </button>
    </aside>
  );
}
