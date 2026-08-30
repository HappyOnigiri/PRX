import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { Snapshot } from "../src/gen/prx/v1/prx_pb";
import { FeatureWorkspace } from "../src/views/FeatureWorkspace";
import {
  makeDependency,
  makeDocument,
  makeFeature,
  makePullRequest,
  makeSnapshot,
  makeTask,
} from "./factories";

const workspaceMocks = vi.hoisted(() => ({
  navigate: vi.fn().mockResolvedValue(undefined),
  snapshot: { data: undefined as Snapshot | undefined, isPending: false },
  hookIndex: 0,
  mutations: Array.from({ length: 4 }, () => ({
    mutate: vi.fn(),
    mutateAsync: vi.fn().mockResolvedValue({}),
    isPending: false,
    error: null as Error | null,
  })),
  api: {
    sync: vi.fn(),
    updateFeature: vi.fn(),
    deleteFeature: vi.fn(),
    addDocument: vi.fn(),
    deleteDocument: vi.fn(),
    getDocument: vi.fn(),
    updateDocument: vi.fn(),
  },
  previewDocument: {
    id: "preview-doc",
    kind: 3,
    title: "Preview document",
    locator: "docs/preview.md",
    isImplementationPlan: false,
  },
}));

vi.mock("@tanstack/react-router", () => ({
  useNavigate: () => workspaceMocks.navigate,
  useParams: () => ({ featureId: "feature-1" }),
}));
vi.mock("../src/api", () => ({
  mutations: workspaceMocks.api,
}));
vi.mock("../src/hooks", () => ({
  useSnapshot: () => workspaceMocks.snapshot,
  useDomainMutation: (mutationFn: (input: unknown) => unknown) => {
    workspaceMocks.hookIndex++;
    const index =
      mutationFn === workspaceMocks.api.addDocument
        ? 1
        : mutationFn === workspaceMocks.api.deleteDocument
          ? 2
          : mutationFn === workspaceMocks.api.updateDocument
            ? 3
            : 0;
    const mutation = workspaceMocks.mutations[index];
    if (!mutation) throw new Error("mutation mock missing");
    mutation.mutate.mockImplementation((input: unknown) => {
      return mutationFn(input);
    });
    return mutation;
  },
}));
vi.mock("../src/views/FeatureGraph", () => ({
  FeatureGraph: ({
    onCreateTask,
    onEditTask,
    onPreviewDocument,
    readOnly,
  }: {
    onCreateTask: () => void;
    onEditTask: (taskId: string) => void;
    onPreviewDocument: (document: unknown) => void;
    readOnly: boolean;
  }) => (
    <div data-testid="feature-graph">
      <span>{readOnly ? "Mock read-only graph" : "Mock active graph"}</span>
      {!readOnly && <button onClick={onCreateTask}>Mock create task</button>}
      <button
        onClick={() => {
          onEditTask("task-1");
        }}
      >
        Mock edit task
      </button>
      <button
        onClick={() => {
          onPreviewDocument(workspaceMocks.previewDocument);
        }}
      >
        Mock preview document
      </button>
    </div>
  ),
}));
vi.mock("../src/views/TaskInspector", () => ({
  TaskInspector: ({
    onClose,
    readOnly,
  }: {
    onClose: () => void;
    readOnly: boolean;
  }) => (
    <aside aria-label="Mock task inspector">
      <span>
        {readOnly ? "Mock read-only inspector" : "Mock active inspector"}
      </span>
      <button onClick={onClose}>Mock close inspector</button>
    </aside>
  ),
}));
vi.mock("../src/views/MarkdownPreview", () => ({
  MarkdownPreview: ({
    document,
    onClose,
  }: {
    document: { title: string };
    onClose: () => void;
  }) => (
    <div role="dialog" aria-label="Mock Markdown preview">
      <p>{document.title}</p>
      <button onClick={onClose}>Mock close preview</button>
    </div>
  ),
}));
vi.mock("../src/views/CreateTaskDialog", () => ({
  CreateTaskDialog: ({ onClose }: { onClose: () => void }) => (
    <div role="dialog" aria-label="Mock create task">
      <button onClick={onClose}>Mock close task</button>
    </div>
  ),
}));
vi.mock("../src/views/EditFeatureDialog", () => ({
  EditFeatureDialog: ({
    onClose,
    onDeleted,
  }: {
    onClose: () => void;
    onDeleted: () => void;
  }) => (
    <div role="dialog" aria-label="Mock edit feature">
      <button onClick={onClose}>Mock close feature edit</button>
      <button onClick={onDeleted}>Mock delete feature</button>
    </div>
  ),
}));

const feature = makeFeature({
  id: "feature-1",
  title: "Payments rollout",
  description: "Ship payments",
  archived: false,
});
const task = makeTask({ id: "task-1", featureId: "feature-1" });
const otherTask = makeTask({ id: "task-2", featureId: "other-feature" });

function populatedSnapshot() {
  return makeSnapshot({
    features: [feature],
    tasks: [task, otherTask],
    dependencies: [
      makeDependency({ blockerTaskId: task.id, blockedTaskId: task.id }),
      makeDependency({ blockerTaskId: otherTask.id, blockedTaskId: task.id }),
    ],
    pullRequests: [makePullRequest({ taskId: task.id })],
    documents: [
      makeDocument({ taskId: task.id }),
      makeDocument({ id: "feature-doc", featureId: feature.id, taskId: "" }),
    ],
  });
}

function mutationAt(index: number) {
  const mutation = workspaceMocks.mutations[index];
  if (!mutation) throw new Error(`mutation mock missing at ${index}`);
  return mutation;
}

describe("FeatureWorkspace", () => {
  afterEach(() => {
    cleanup();
    vi.restoreAllMocks();
  });

  beforeEach(() => {
    workspaceMocks.hookIndex = 0;
    workspaceMocks.snapshot.data = populatedSnapshot();
    workspaceMocks.snapshot.isPending = false;
    workspaceMocks.navigate.mockClear();
    for (const mutation of workspaceMocks.mutations) {
      mutation.mutate.mockReset();
      mutation.mutateAsync.mockReset();
      mutation.mutateAsync.mockResolvedValue({});
      mutation.isPending = false;
      mutation.error = null;
    }
  });

  it("coordinates active sync, feature management, task inspection, and previews", () => {
    render(<FeatureWorkspace />);

    expect(
      screen.getByRole("heading", { name: "Payments rollout" }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: "Copy Feature ID" }),
    ).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Sync GitHub" }));
    expect(mutationAt(0).mutate).toHaveBeenCalledWith("feature-1");
    fireEvent.click(screen.getByRole("button", { name: "Edit feature" }));
    expect(
      screen.getByRole("dialog", { name: "Mock edit feature" }),
    ).toBeInTheDocument();
    fireEvent.click(
      screen.getByRole("button", { name: "Mock close feature edit" }),
    );
    expect(
      screen.queryByRole("dialog", { name: "Mock edit feature" }),
    ).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Mock create task" }));
    expect(
      screen.getByRole("dialog", { name: "Mock create task" }),
    ).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Mock close task" }));
    fireEvent.click(screen.getByRole("button", { name: "Mock edit task" }));
    expect(
      screen.getByRole("complementary", { name: "Mock task inspector" }),
    ).toBeInTheDocument();
    fireEvent.click(
      screen.getByRole("button", { name: "Mock close inspector" }),
    );
    fireEvent.click(
      screen.getByRole("button", { name: "Mock preview document" }),
    );
    expect(
      screen.getByRole("dialog", { name: "Mock Markdown preview" }),
    ).toHaveTextContent("Preview document");
    fireEvent.click(screen.getByRole("button", { name: "Mock close preview" }));

    fireEvent.click(screen.getByRole("button", { name: "Edit feature" }));
    fireEvent.click(
      screen.getByRole("button", { name: "Mock delete feature" }),
    );
    expect(workspaceMocks.navigate).toHaveBeenCalledWith({ to: "/" });
  });

  it("renders loading, missing-feature, and sync-pending states", () => {
    workspaceMocks.snapshot.isPending = true;
    const { rerender } = render(<FeatureWorkspace />);
    expect(
      screen.getByRole("heading", { name: "Tracing feature graph…" }),
    ).toBeInTheDocument();

    workspaceMocks.snapshot.isPending = false;
    workspaceMocks.snapshot.data = makeSnapshot({ features: [] });
    rerender(<FeatureWorkspace />);
    expect(
      screen.getByRole("heading", { name: "Feature not found" }),
    ).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Return to overview" }));
    expect(workspaceMocks.navigate).toHaveBeenCalledWith({ to: "/" });

    workspaceMocks.snapshot.data = populatedSnapshot();
    mutationAt(0).isPending = true;
    rerender(<FeatureWorkspace />);
    expect(screen.getByRole("button", { name: "Syncing…" })).toBeDisabled();
  });

  it("makes archived workspaces read-only and returns deletion to the archive", () => {
    workspaceMocks.snapshot.data = makeSnapshot({
      features: [{ ...feature, archived: true }],
      tasks: [task],
    });
    render(<FeatureWorkspace />);

    expect(screen.getByText("Archived · read-only")).toBeInTheDocument();
    expect(screen.getByText("Mock read-only graph")).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: "Sync GitHub" }),
    ).not.toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: "Mock create task" }),
    ).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Mock edit task" }));
    expect(screen.getByText("Mock read-only inspector")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Manage feature" }));
    fireEvent.click(
      screen.getByRole("button", { name: "Mock delete feature" }),
    );
    expect(workspaceMocks.navigate).toHaveBeenCalledWith({ to: "/archived" });
  });
});
