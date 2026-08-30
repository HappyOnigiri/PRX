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
  PullRequestDisplayState,
  TaskKind,
  TaskStatus,
} from "../src/gen/prx/v1/prx_pb";
import { TaskInspector } from "../src/views/TaskInspector";
import { makeDocument, makePullRequest, makeTask } from "./factories";

const inspectorMocks = vi.hoisted(() => ({
  hookIndex: 0,
  api: {
    updateTask: vi.fn(),
    deleteTask: vi.fn(),
    attachPR: vi.fn(),
    detachPR: vi.fn(),
    addDocument: vi.fn(),
    deleteDocument: vi.fn(),
    getDocument: vi.fn(),
    updateDocument: vi.fn(),
  },
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
    mutation.mutate.mockImplementation(
      (input: unknown, options?: { onSuccess?: (data: unknown) => void }) => {
        const result = mutationFn(input);
        if (options?.onSuccess) {
          void Promise.resolve(result).then(options.onSuccess);
        }
        return result;
      },
    );
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

  it("edits a task and manages its pull request and references", async () => {
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
    const pullRequest = makePullRequest({
      taskId: task.id,
      stale: true,
      syncError: "GitHub data is old",
      displayState: PullRequestDisplayState.MERGED,
    });
    const markdown = makeDocument({
      id: "document-md",
      taskId: task.id,
      kind: DocumentKind.LOCAL_FILE,
      title: "Delivery plan",
      locator: "docs/delivery.md",
    });
    const url = makeDocument({
      id: "document-url",
      taskId: task.id,
      kind: DocumentKind.URL,
      title: "Runbook",
    });
    const inline = makeDocument({
      id: "document-inline",
      taskId: task.id,
      kind: DocumentKind.MARKDOWN,
      title: "Inline plan",
      locator: "",
      isImplementationPlan: true,
    });
    const onClose = vi.fn();
    const onPreview = vi.fn();
    render(
      <TaskInspector
        task={task}
        tasks={[task, blocker]}
        pullRequest={pullRequest}
        documents={[markdown, url, inline]}
        onPreview={onPreview}
        onClose={onClose}
      />,
    );

    expect(
      screen.getByRole("button", { name: "Copy Task ID" }),
    ).toBeInTheDocument();
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
    expect(
      screen.queryByRole("combobox", { name: "Blocker task" }),
    ).not.toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: /^Add$/ }),
    ).not.toBeInTheDocument();
    expect(
      screen.queryByRole("heading", { name: "Blocked by" }),
    ).not.toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: "Remove dependency" }),
    ).not.toBeInTheDocument();

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

    fireEvent.click(
      screen.getByRole("button", { name: "Delivery plandocs/delivery.md" }),
    );
    expect(onPreview).toHaveBeenCalledWith(markdown);
    fireEvent.click(screen.getByRole("button", { name: "Delete Runbook" }));
    expect(mutationAt(5).mutate).toHaveBeenCalledWith("document-url");

    inspectorMocks.api.getDocument.mockResolvedValue({
      content: "# Old\n\n- first\n- second",
    });
    fireEvent.click(screen.getByRole("button", { name: "Edit Inline plan" }));
    expect(
      await screen.findByRole("textbox", { name: "Edit Inline plan" }),
    ).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Cancel edit" }));
    expect(
      screen.queryByRole("textbox", { name: "Edit Inline plan" }),
    ).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Edit Inline plan" }));
    const markdownEditor = await screen.findByRole("textbox", {
      name: "Edit Inline plan",
    });
    expect(markdownEditor).toHaveValue("# Old\n\n- first\n- second");
    fireEvent.change(markdownEditor, {
      target: { value: "# New\n\n- first\n- second" },
    });
    const markdownEditForm = markdownEditor.closest("form");
    if (!markdownEditForm) throw new Error("markdown edit form missing");
    fireEvent.submit(markdownEditForm);
    expect(inspectorMocks.api.updateDocument).toHaveBeenCalledWith({
      id: "document-inline",
      source: { case: "markdown", value: "# New\n\n- first\n- second" },
    });

    const referenceKind = screen.getAllByRole("combobox").at(-1);
    if (!referenceKind) throw new Error("reference kind select missing");
    fireEvent.change(referenceKind, {
      target: { value: String(DocumentKind.LOCAL_FILE) },
    });
    const referenceValue = screen.getByPlaceholderText("docs/plan.md");
    fireEvent.change(referenceValue, {
      target: { value: "docs/new.md" },
    });
    fireEvent.change(screen.getByPlaceholderText("Design notes"), {
      target: { value: "New plan" },
    });
    const referenceForm = referenceValue.closest("form");
    if (!referenceForm) throw new Error("reference form missing");
    fireEvent.submit(referenceForm);
    expect(inspectorMocks.api.addDocument).toHaveBeenCalledWith({
      taskId: task.id,
      kind: DocumentKind.LOCAL_FILE,
      title: "New plan",
      value: "docs/new.md",
      isImplementationPlan: false,
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

  it("reports a failed Markdown read for editing", () => {
    const task = makeTask();
    const inline = makeDocument({
      id: "document-inline",
      taskId: task.id,
      kind: DocumentKind.MARKDOWN,
      title: "Inline plan",
      locator: "",
    });
    const inspector = (
      <TaskInspector
        task={task}
        tasks={[task]}
        pullRequest={undefined}
        documents={[inline]}
        onPreview={vi.fn()}
        onClose={vi.fn()}
      />
    );
    mutationAt(7).error = new Error("read failed");
    render(inspector);

    expect(screen.getByRole("alert")).toHaveTextContent("read failed");
  });

  it("does not close when task deletion fails", async () => {
    const onClose = vi.fn();
    mutationAt(0).mutateAsync.mockRejectedValueOnce(new Error("delete failed"));
    render(
      <TaskInspector
        task={makeTask()}
        tasks={[makeTask()]}
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

  it("keeps archived task content readable without mutation controls", () => {
    const task = makeTask({
      title: "Archived task",
      scope: "Historical scope",
      assignee: "Ren",
    });
    const blocker = makeTask({ id: "task-2", title: "Archived blocker" });
    const markdown = makeDocument({
      kind: DocumentKind.LOCAL_FILE,
      title: "Decision log",
      locator: "docs/decision.md",
    });
    const onPreview = vi.fn();
    render(
      <TaskInspector
        task={task}
        tasks={[task, blocker]}
        pullRequest={makePullRequest({ taskId: task.id })}
        documents={[markdown]}
        onPreview={onPreview}
        onClose={vi.fn()}
        readOnly
      />,
    );

    expect(screen.getByText("Archived task · read-only")).toBeInTheDocument();
    expect(screen.getByText("Historical scope")).toBeInTheDocument();
    expect(screen.queryByText("Archived blocker")).not.toBeInTheDocument();
    expect(screen.getByRole("link", { name: "acme/prx #42" })).toHaveAttribute(
      "target",
      "_blank",
    );
    fireEvent.click(
      screen.getByRole("button", { name: "Decision logdocs/decision.md" }),
    );
    expect(onPreview).toHaveBeenCalledWith(markdown);
    expect(screen.queryByRole("textbox")).not.toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: "Detach" }),
    ).not.toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: "Add" }),
    ).not.toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: "Delete task and references" }),
    ).not.toBeInTheDocument();
  });
});
