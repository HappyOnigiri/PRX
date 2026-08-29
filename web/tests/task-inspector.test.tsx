import { Code, ConnectError } from "@connectrpc/connect";
import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import {
  BlockedReasonCode,
  DocumentKind,
  DomainErrorCode,
  ErrorDetailSchema,
  PullRequestDisplayState,
  TaskKind,
  TaskStatus,
} from "../src/gen/prx/v1/prx_pb";
import { TaskInspector } from "../src/views/TaskInspector";
import {
  makeDependency,
  makeDocument,
  makePullRequest,
  makeTask,
} from "./factories";

const inspectorMocks = vi.hoisted(() => ({
  api: {
    updateTask: vi.fn(),
    deleteTask: vi.fn(),
    getImplementationPlan: vi.fn(),
    upsertImplementationPlan: vi.fn(),
    deleteImplementationPlan: vi.fn(),
    attachPR: vi.fn(),
    detachPR: vi.fn(),
    addDependency: vi.fn(),
    removeDependency: vi.fn(),
    addDocument: vi.fn(),
    deleteDocument: vi.fn(),
  },
  hookIndex: 0,
  mutations: Array.from({ length: 10 }, () => ({
    mutate: vi.fn(),
    mutateAsync: vi.fn().mockResolvedValue({}),
    isPending: false,
    error: null as Error | null,
  })),
}));

vi.mock("../src/api", () => ({ mutations: inspectorMocks.api }));
vi.mock("../src/hooks", () => ({
  useDomainMutation: (mutationFn: (input: unknown) => unknown) => {
    const mutation =
      inspectorMocks.mutations[
        inspectorMocks.hookIndex++ % inspectorMocks.mutations.length
      ];
    if (!mutation) throw new Error("mutation mock missing");
    mutation.mutate.mockImplementation((input: unknown) => {
      return mutationFn(input);
    });
    return mutation;
  },
}));

function mutationAt(index: number) {
  const mutation = inspectorMocks.mutations[index];
  if (!mutation) throw new Error(`mutation mock missing at ${index}`);
  return mutation;
}

describe("TaskInspector", () => {
  afterEach(() => {
    cleanup();
    vi.restoreAllMocks();
  });

  beforeEach(() => {
    inspectorMocks.hookIndex = 0;
    for (const mutation of inspectorMocks.mutations) {
      mutation.mutate.mockReset();
      mutation.mutateAsync.mockReset();
      mutation.mutateAsync.mockResolvedValue({});
      mutation.isPending = false;
      mutation.error = null;
    }
  });

  it("edits a task and manages its pull request, dependencies, and references", async () => {
    const task = makeTask({
      title: "Current task",
      scope: "Initial scope",
      kind: TaskKind.MANUAL,
      status: TaskStatus.AUTO,
      assignee: "Mika",
      blockedReason: {
        code: BlockedReasonCode.WAITING_FOR_BLOCKER,
        blockerTaskId: "task-2",
      },
    });
    const blocker = makeTask({ id: "task-2", title: "Blocker task" });
    const candidate = makeTask({ id: "task-3", title: "Candidate task" });
    const dependency = makeDependency({
      blockerTaskId: "task-2",
      blockedTaskId: task.id,
    });
    const pullRequest = makePullRequest({
      taskId: task.id,
      stale: true,
      syncError: "GitHub data is old",
      displayState: PullRequestDisplayState.MERGED,
    });
    const markdown = makeDocument({
      id: "document-md",
      taskId: task.id,
      kind: DocumentKind.MARKDOWN_PATH,
      title: "Delivery plan",
      value: "docs/delivery.md",
    });
    const url = makeDocument({
      id: "document-url",
      taskId: task.id,
      kind: DocumentKind.URL,
      title: "Runbook",
    });
    const onClose = vi.fn();
    const onPreview = vi.fn();
    mutationAt(6).error = new ConnectError(
      "cycle would be introduced",
      Code.FailedPrecondition,
      undefined,
      [
        {
          desc: ErrorDetailSchema,
          value: {
            code: DomainErrorCode.CYCLE,
            path: ["task-2", task.id],
          },
        },
      ],
    );

    render(
      <TaskInspector
        task={task}
        tasks={[task, blocker, candidate]}
        dependencies={[dependency]}
        pullRequest={pullRequest}
        documents={[markdown, url]}
        onPreview={onPreview}
        onClose={onClose}
      />,
    );

    expect(screen.getByText("Waiting for Blocker task")).toBeInTheDocument();
    expect(screen.getByText("GitHub data is old")).toBeInTheDocument();
    expect(screen.getByText("merged")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "acme/prx #42" })).toHaveAttribute(
      "target",
      "_blank",
    );
    expect(screen.getByRole("link", { name: /Runbook/ })).toHaveAttribute(
      "href",
      "https://example.com/runbook",
    );
    expect(
      screen.getByRole("button", { name: "Delete Runbook" }),
    ).toBeInTheDocument();
    expect(screen.getByRole("alert")).toHaveTextContent("Blocker task");

    fireEvent.change(screen.getByLabelText("Title"), {
      target: { value: "Updated task" },
    });
    fireEvent.change(screen.getByLabelText("Scope"), {
      target: { value: "Updated scope" },
    });
    fireEvent.change(screen.getByLabelText("Status"), {
      target: { value: String(TaskStatus.CLOSED) },
    });
    fireEvent.change(screen.getByLabelText("Assignee"), {
      target: { value: "Ren" },
    });
    fireEvent.click(screen.getByRole("button", { name: "Save task" }));
    expect(mutationAt(1).mutate).toHaveBeenCalledWith({
      id: task.id,
      title: "Updated task",
      scope: "Updated scope",
      status: TaskStatus.CLOSED,
      assignee: "Ren",
    });

    fireEvent.click(screen.getByRole("button", { name: "Detach" }));
    expect(mutationAt(3).mutate).toHaveBeenCalledWith(task.id);
    fireEvent.click(screen.getByRole("button", { name: "Remove dependency" }));
    expect(mutationAt(7).mutate).toHaveBeenCalledWith({
      blocker: "task-2",
      blocked: task.id,
    });

    fireEvent.change(screen.getByRole("combobox", { name: "Blocker task" }), {
      target: { value: "task-3" },
    });
    fireEvent.click(screen.getByRole("button", { name: "Add" }));
    expect(mutationAt(6).mutate).toHaveBeenCalledWith({
      blocker: "task-3",
      blocked: task.id,
    });

    fireEvent.click(
      screen.getByRole("button", { name: "Delivery plandocs/delivery.md" }),
    );
    expect(onPreview).toHaveBeenCalledWith(markdown);
    fireEvent.click(screen.getByRole("button", { name: "Delete Runbook" }));
    expect(mutationAt(9).mutate).toHaveBeenCalledWith("document-url");

    const referenceValue = screen.getByPlaceholderText(
      "https://… or docs/plan.md",
    );
    fireEvent.change(referenceValue, {
      target: { value: "docs/new.md" },
    });
    fireEvent.change(screen.getByPlaceholderText("Design notes"), {
      target: { value: "New plan" },
    });
    const referenceKind = screen.getAllByRole("combobox").at(-1);
    if (!referenceKind) throw new Error("reference kind select missing");
    fireEvent.change(referenceKind, {
      target: { value: String(DocumentKind.MARKDOWN_PATH) },
    });
    fireEvent.click(screen.getByRole("button", { name: "Add reference" }));
    expect(mutationAt(8).mutate).toHaveBeenCalledWith({
      taskId: task.id,
      kind: DocumentKind.MARKDOWN_PATH,
      title: "New plan",
      value: "docs/new.md",
    });

    vi.spyOn(window, "confirm").mockReturnValue(true);
    fireEvent.click(
      screen.getByRole("button", { name: "Delete task and references" }),
    );
    await waitFor(() => {
      expect(onClose).toHaveBeenCalledOnce();
    });
    expect(mutationAt(0).mutateAsync).toHaveBeenCalledWith(task.id);
  });

  it("attaches a pull request when none is linked and closes explicitly", () => {
    const task = makeTask({ kind: TaskKind.PULL_REQUEST });
    render(
      <TaskInspector
        task={task}
        tasks={[task]}
        dependencies={[]}
        pullRequest={undefined}
        documents={[]}
        onPreview={vi.fn()}
        onClose={vi.fn()}
      />,
    );

    fireEvent.change(
      screen.getByPlaceholderText("https://github.com/org/repo/pull/42"),
      {
        target: { value: "https://github.com/acme/prx/pull/99" },
      },
    );
    fireEvent.click(screen.getByRole("button", { name: "Attach" }));
    expect(mutationAt(2).mutate).toHaveBeenCalledWith({
      taskId: task.id,
      url: "https://github.com/acme/prx/pull/99",
    });
    expect(
      screen.getByRole("option", { name: "Completed" }),
    ).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Close inspector" }));
  });

  it("does not close when task deletion fails", async () => {
    const onClose = vi.fn();
    mutationAt(0).mutateAsync.mockRejectedValueOnce(new Error("delete failed"));
    render(
      <TaskInspector
        task={makeTask()}
        tasks={[makeTask()]}
        dependencies={[]}
        pullRequest={undefined}
        documents={[]}
        onPreview={vi.fn()}
        onClose={onClose}
      />,
    );
    vi.spyOn(window, "confirm").mockReturnValue(true);

    fireEvent.click(
      screen.getByRole("button", { name: "Delete task and references" }),
    );
    await waitFor(() => {
      expect(mutationAt(0).mutateAsync).toHaveBeenCalledOnce();
    });
    expect(onClose).not.toHaveBeenCalled();
  });
});
