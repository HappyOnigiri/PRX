import { create } from "@bufbuild/protobuf";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import type { PropsWithChildren } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import {
  BlockedReasonCode,
  CheckState,
  DocumentKind,
  PullRequestDisplayState,
  TaskBlockLabel,
  TaskLabelAppearanceSchema,
  TaskLabelAppearancesSchema,
  TaskStatus,
} from "../src/gen/prx/v1/prx_pb";
import { TaskInspector } from "../src/views/TaskInspector";
import { makeDocument, makePullRequest, makeTask } from "./factories";

// 資料の編集ダイアログはファイル選択に react-query をそのまま使うので、
// インスペクタの描画にも Provider を添える。
function Wrapper({ children }: PropsWithChildren) {
  const queryClient = new QueryClient({
    defaultOptions: { mutations: { retry: false }, queries: { retry: false } },
  });
  return (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );
}

type ApiName =
  | "updateTask"
  | "deleteTask"
  | "attachPR"
  | "detachPR"
  | "addDocument"
  | "deleteDocument"
  | "getDocument"
  | "updateDocument";

const inspectorMocks = vi.hoisted(() => {
  const names: string[] = [
    "updateTask",
    "deleteTask",
    "attachPR",
    "detachPR",
    "addDocument",
    "deleteDocument",
    "getDocument",
    "updateDocument",
  ];
  const api: Record<string, ReturnType<typeof vi.fn>> = {};
  const mutations: Record<
    string,
    {
      mutate: ReturnType<typeof vi.fn>;
      mutateAsync: ReturnType<typeof vi.fn>;
      isPending: boolean;
      error: Error | null;
    }
  > = {};
  for (const name of names) {
    api[name] = vi.fn();
    mutations[name] = {
      mutate: vi.fn(),
      mutateAsync: vi.fn().mockResolvedValue({}),
      isPending: false,
      error: null,
    };
  }
  return { api, mutations, names, selectLocalFile: vi.fn() };
});

const apiNames = inspectorMocks.names as ApiName[];

vi.mock("../src/api", () => ({
  mutations: inspectorMocks.api,
  selectLocalFile: inspectorMocks.selectLocalFile,
}));
vi.mock("../src/hooks", () => ({
  useDomainMutation: (mutationFn: (input: unknown) => unknown) => {
    // attachPR だけは呼び出し側が引数を組み替える無名関数を渡すので、
    // api の関数と一致しないものはそこへ寄せる。
    const name =
      apiNames.find((entry) => inspectorMocks.api[entry] === mutationFn) ??
      "attachPR";
    const mutation = inspectorMocks.mutations[name];
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

function apiFor(name: ApiName) {
  const fn = inspectorMocks.api[name];
  if (!fn) throw new Error(`api mock missing for ${name}`);
  return fn;
}

function mutationFor(name: ApiName) {
  const mutation = inspectorMocks.mutations[name];
  if (!mutation) throw new Error(`mutation mock missing for ${name}`);
  return mutation;
}

describe("TaskInspector", () => {
  afterEach(() => {
    cleanup();
    vi.restoreAllMocks();
  });

  beforeEach(() => {
    for (const name of apiNames) inspectorMocks.api[name]?.mockReset();
    for (const mutation of Object.values(inspectorMocks.mutations)) {
      mutation.mutate.mockReset();
      mutation.mutateAsync.mockReset();
      mutation.mutateAsync.mockResolvedValue({});
      mutation.isPending = false;
      mutation.error = null;
    }
  });

  it("edits a task and keeps reference editing in the inspector", async () => {
    const task = makeTask({
      title: "Current task",
      scope: "Initial scope",
      status: TaskStatus.NOT_STARTED,
      assignee: "Bob",
      blockedReason: {
        code: BlockedReasonCode.WAITING_FOR_BLOCKER,
        blockerTaskId: "task-2",
      },
      blockLabels: [TaskBlockLabel.DEPENDENCY_UNRESOLVED],
    });
    const blocker = makeTask({ id: "task-2", title: "Blocker task" });
    const pullRequest = makePullRequest({
      taskId: task.id,
      stale: true,
      syncError: "GitHub data is old",
      displayState: PullRequestDisplayState.MERGED,
      checkState: CheckState.FAILURE,
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
    const appearances = create(TaskLabelAppearancesSchema, {
      values: [
        create(TaskLabelAppearanceSchema, {
          key: "status.in_progress",
          text: "Coding",
          textOverridden: true,
        }),
      ],
    });
    render(
      <TaskInspector
        task={task}
        tasks={[task, blocker]}
        pullRequest={pullRequest}
        documents={[markdown, url, inline]}
        appearances={appearances}
        onPreview={onPreview}
        onClose={onClose}
      />,
      { wrapper: Wrapper },
    );

    const taskIdButton = screen.getByRole("button", { name: "Copy Task ID" });
    expect(taskIdButton).toHaveTextContent(task.id);
    expect(taskIdButton.querySelector("svg")).not.toBeInTheDocument();
    expect(screen.queryByText("Task ID")).not.toBeInTheDocument();
    // ブロックラベルは短い語で出し、どの blocker を待つかは可視のテキストで
    // 添える（title はホバーできる環境向けの補足）。
    expect(screen.getByText("dependency")).toHaveAttribute(
      "title",
      "Waiting for Blocker task",
    );
    expect(
      screen.getByText("Waiting for Blocker task", {
        selector: ".inspector-blocks-detail",
      }),
    ).toBeInTheDocument();
    expect(screen.getByText("GitHub data is old")).toBeInTheDocument();
    expect(screen.getByText("merged")).toBeInTheDocument();
    expect(screen.getByText("failing")).toBeInTheDocument();
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
      screen.queryByRole("button", { name: "Add reference" }),
    ).not.toBeInTheDocument();
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
      target: { value: "Carol" },
    });
    fireEvent.click(screen.getByRole("button", { name: "Save task" }));
    expect(mutationFor("updateTask").mutate).toHaveBeenCalledWith({
      id: task.id,
      title: "Updated task",
      scope: "Updated scope",
      status: TaskStatus.CLOSED,
      assignee: "Carol",
    });

    fireEvent.click(screen.getByRole("button", { name: "Detach" }));
    expect(mutationFor("detachPR").mutate).toHaveBeenCalledWith(task.id);

    fireEvent.click(
      screen.getByRole("button", { name: "Delivery plandocs/delivery.md" }),
    );
    expect(onPreview).toHaveBeenCalledWith(markdown);
    apiFor("getDocument").mockResolvedValue({
      document: inline,
      content: "# Old\n\n- first\n- second",
    });
    fireEvent.click(screen.getByRole("button", { name: "Edit Inline plan" }));
    const editDialog = await screen.findByRole("dialog", {
      name: "Edit task reference",
    });
    fireEvent.click(screen.getByRole("button", { name: "Cancel" }));
    expect(editDialog).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Edit Inline plan" }));
    await screen.findByRole("dialog", { name: "Edit task reference" });
    const markdownEditor = screen.getByLabelText("Markdown content");
    expect(markdownEditor).toHaveValue("# Old\n\n- first\n- second");
    fireEvent.change(markdownEditor, {
      target: { value: "# New\n\n- first\n- second" },
    });
    fireEvent.change(screen.getByLabelText("Reference title (optional)"), {
      target: { value: "Inline plan v2" },
    });
    fireEvent.click(screen.getByRole("button", { name: "Save" }));
    await waitFor(() => {
      expect(mutationFor("updateDocument").mutateAsync).toHaveBeenCalledWith({
        id: "document-inline",
        title: "Inline plan v2",
        source: { case: "markdown", value: "# New\n\n- first\n- second" },
        isImplementationPlan: true,
      });
    });

    vi.spyOn(window, "confirm").mockReturnValue(true);
    fireEvent.click(
      screen.getByRole("button", { name: "Delete task and references" }),
    );
    await waitFor(() => {
      expect(onClose).toHaveBeenCalledOnce();
    });
    expect(mutationFor("deleteTask").mutateAsync).toHaveBeenCalledWith(task.id);
  });

  it("attaches a pull request when none is linked and closes explicitly", () => {
    const task = makeTask();
    render(
      <TaskInspector
        task={task}
        tasks={[task]}
        pullRequest={undefined}
        documents={[]}
        onPreview={vi.fn()}
        onClose={vi.fn()}
      />,
      { wrapper: Wrapper },
    );

    fireEvent.change(
      screen.getByPlaceholderText("https://github.com/org/repo/pull/42"),
      {
        target: { value: "https://github.com/acme/prx/pull/99" },
      },
    );
    fireEvent.click(screen.getByRole("button", { name: "Attach" }));
    expect(mutationFor("attachPR").mutate).toHaveBeenCalledWith({
      taskId: task.id,
      url: "https://github.com/acme/prx/pull/99",
    });
    expect(
      screen.getByRole("option", { name: "Completed" }),
    ).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Close inspector" }));
  });

  it("warns before dropping unsaved task fields", () => {
    const task = makeTask();
    const onClose = vi.fn();
    render(
      <TaskInspector
        task={task}
        tasks={[task]}
        pullRequest={undefined}
        documents={[]}
        onPreview={vi.fn()}
        onClose={onClose}
      />,
      { wrapper: Wrapper },
    );

    expect(screen.getByRole("button", { name: "Save task" })).toBeDisabled();
    fireEvent.click(screen.getByRole("button", { name: "Close inspector" }));
    expect(onClose).toHaveBeenCalledOnce();

    onClose.mockClear();
    fireEvent.change(screen.getByLabelText("Assignee"), {
      target: { value: "Carol" },
    });
    expect(screen.getByRole("button", { name: "Save task" })).toBeEnabled();
    fireEvent.click(screen.getByRole("button", { name: "Close inspector" }));
    expect(onClose).not.toHaveBeenCalled();
    expect(
      screen.getByRole("dialog", { name: "Discard unsaved changes?" }),
    ).toBeInTheDocument();
    // 確認をやめたら編集は残り、インスペクタも開いたままになる。
    fireEvent.click(screen.getByRole("button", { name: "Cancel" }));
    expect(screen.getByLabelText("Assignee")).toHaveValue("Carol");
    expect(onClose).not.toHaveBeenCalled();

    fireEvent.click(screen.getByRole("button", { name: "Close inspector" }));
    fireEvent.click(screen.getByRole("button", { name: "Discard" }));
    expect(onClose).toHaveBeenCalledOnce();
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
    mutationFor("getDocument").error = new Error("read failed");
    render(inspector, { wrapper: Wrapper });

    expect(screen.getByRole("alert")).toHaveTextContent("read failed");
  });

  it("does not close when task deletion fails", async () => {
    const onClose = vi.fn();
    mutationFor("deleteTask").mutateAsync.mockRejectedValueOnce(
      new Error("delete failed"),
    );
    render(
      <TaskInspector
        task={makeTask()}
        tasks={[makeTask()]}
        pullRequest={undefined}
        documents={[]}
        onPreview={vi.fn()}
        onClose={onClose}
      />,
      { wrapper: Wrapper },
    );
    vi.spyOn(window, "confirm").mockReturnValue(true);

    fireEvent.click(
      screen.getByRole("button", { name: "Delete task and references" }),
    );
    await waitFor(() => {
      expect(mutationFor("deleteTask").mutateAsync).toHaveBeenCalledOnce();
    });
    expect(onClose).not.toHaveBeenCalled();
  });

  it("keeps archived task content readable without mutation controls", () => {
    const task = makeTask({
      title: "Archived task",
      scope: "Historical scope",
      assignee: "Carol",
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
      { wrapper: Wrapper },
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
    expect(
      screen.queryByRole("button", { name: "Delete Decision log" }),
    ).not.toBeInTheDocument();
  });

  it("confirms before deleting a reference from the inspector", () => {
    apiFor("deleteDocument").mockClear();
    const task = makeTask();
    const markdown = makeDocument({
      id: "document-md",
      taskId: task.id,
      kind: DocumentKind.LOCAL_FILE,
      title: "Delivery plan",
      locator: "docs/delivery.md",
    });
    render(
      <TaskInspector
        task={task}
        tasks={[task]}
        pullRequest={undefined}
        documents={[markdown]}
        onPreview={vi.fn()}
        onClose={vi.fn()}
      />,
      { wrapper: Wrapper },
    );

    fireEvent.click(
      screen.getByRole("button", { name: "Delete Delivery plan" }),
    );
    expect(
      screen.getByRole("dialog", { name: "Delete Delivery plan?" }),
    ).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Cancel" }));
    expect(apiFor("deleteDocument")).not.toHaveBeenCalled();
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();

    fireEvent.click(
      screen.getByRole("button", { name: "Delete Delivery plan" }),
    );
    fireEvent.click(screen.getByRole("button", { name: "Delete reference" }));
    expect(apiFor("deleteDocument")).toHaveBeenCalledWith("document-md");
  });
});
