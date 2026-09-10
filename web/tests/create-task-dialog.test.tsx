import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { CreateTaskDialog } from "../src/views/CreateTaskDialog";

const dialogMocks = vi.hoisted(() => ({
  mutation: {
    mutateAsync: vi.fn(),
    isPending: false,
    error: null as Error | null,
  },
  createTask: vi.fn(),
  updateTask: vi.fn(),
  refreshSnapshot: vi.fn(),
  addDependency: vi.fn(),
}));

vi.mock("../src/api", () => ({
  mutations: {
    createTask: dialogMocks.createTask,
    updateTask: dialogMocks.updateTask,
    addDependency: dialogMocks.addDependency,
  },
}));
// mutateAsync からダイアログが渡した本体を実行して、フォーム送信から作成と依存
// 追加までが 1 つの経路としてつながっていることを見る。
vi.mock("../src/hooks", () => ({
  useDomainMutation: (mutationFn: (input: unknown) => Promise<unknown>) => {
    dialogMocks.mutation.mutateAsync.mockImplementation((input: unknown) =>
      mutationFn(input),
    );
    return dialogMocks.mutation;
  },
  useSnapshotRefresh: () => dialogMocks.refreshSnapshot,
}));

describe("CreateTaskDialog", () => {
  afterEach(cleanup);
  beforeEach(() => {
    dialogMocks.mutation.mutateAsync.mockReset();
    dialogMocks.mutation.isPending = false;
    dialogMocks.mutation.error = null;
    dialogMocks.createTask.mockReset();
    dialogMocks.createTask.mockResolvedValue({ task: { id: "task-new" } });
    dialogMocks.updateTask.mockReset();
    dialogMocks.updateTask.mockResolvedValue({});
    dialogMocks.addDependency.mockReset();
    dialogMocks.addDependency.mockResolvedValue({});
    dialogMocks.refreshSnapshot.mockReset();
    dialogMocks.refreshSnapshot.mockResolvedValue(undefined);
  });

  function submitForm(values: {
    title?: string;
    scope?: string;
    assignee?: string;
  }) {
    for (const [label, value] of [
      ["Title", values.title],
      ["Scope", values.scope],
      ["Assignee", values.assignee],
    ] as const) {
      if (value !== undefined)
        fireEvent.change(screen.getByLabelText(label), { target: { value } });
    }
    fireEvent.submit(screen.getByRole("form", { name: "Create task" }));
  }

  it("converts form values into a task and closes on success", async () => {
    const onClose = vi.fn();
    render(<CreateTaskDialog featureId="feature-1" onClose={onClose} />);

    submitForm({
      title: "Implement checkout",
      scope: "API and acceptance tests",
      assignee: "Carol",
    });

    await waitFor(() => {
      expect(onClose).toHaveBeenCalledOnce();
    });
    expect(dialogMocks.createTask).toHaveBeenCalledWith({
      featureId: "feature-1",
      title: "Implement checkout",
      scope: "API and acceptance tests",
      assignee: "Carol",
    });
    expect(dialogMocks.addDependency).not.toHaveBeenCalled();
  });

  it("adds the dependency after creating the task", async () => {
    const onClose = vi.fn();
    render(
      <CreateTaskDialog
        featureId="feature-1"
        onClose={onClose}
        dependency={{ taskId: "task-1", direction: "blockedBy" }}
        dependencyTitle="Blocker task"
      />,
    );
    expect(
      screen.getByText("The new task will be blocked by Blocker task."),
    ).toBeInTheDocument();

    submitForm({ title: "Implement checkout" });

    await waitFor(() => {
      expect(onClose).toHaveBeenCalledOnce();
    });
    expect(dialogMocks.createTask.mock.invocationCallOrder[0]).toBeLessThan(
      dialogMocks.addDependency.mock.invocationCallOrder[0] ?? 0,
    );
    expect(dialogMocks.addDependency).toHaveBeenCalledWith(
      "task-1",
      "task-new",
    );
  });

  it("reverses the dependency when the new task is the blocker", async () => {
    render(
      <CreateTaskDialog
        featureId="feature-1"
        onClose={vi.fn()}
        dependency={{ taskId: "task-1", direction: "blocks" }}
        dependencyTitle="Blocked task"
      />,
    );
    expect(
      screen.getByText("The new task will block Blocked task."),
    ).toBeInTheDocument();

    submitForm({ title: "Implement checkout" });

    await waitFor(() => {
      expect(dialogMocks.addDependency).toHaveBeenCalledWith(
        "task-new",
        "task-1",
      );
    });
  });

  it("retries only the dependency when the task already exists", async () => {
    dialogMocks.addDependency.mockRejectedValueOnce(new Error("offline"));
    const onClose = vi.fn();
    render(
      <CreateTaskDialog
        featureId="feature-1"
        onClose={onClose}
        dependency={{ taskId: "task-1", direction: "blockedBy" }}
      />,
    );

    submitForm({ title: "Implement checkout" });
    await waitFor(() => {
      expect(dialogMocks.addDependency).toHaveBeenCalledOnce();
    });
    expect(onClose).not.toHaveBeenCalled();

    submitForm({});
    await waitFor(() => {
      expect(onClose).toHaveBeenCalledOnce();
    });
    expect(dialogMocks.createTask).toHaveBeenCalledOnce();
    expect(dialogMocks.addDependency).toHaveBeenCalledTimes(2);
  });

  it("sends edited values as an update when retrying the dependency", async () => {
    dialogMocks.addDependency.mockRejectedValueOnce(new Error("offline"));
    render(
      <CreateTaskDialog
        featureId="feature-1"
        onClose={vi.fn()}
        dependency={{ taskId: "task-1", direction: "blockedBy" }}
      />,
    );

    submitForm({ title: "Implement checkout" });
    await waitFor(() => {
      expect(dialogMocks.addDependency).toHaveBeenCalledOnce();
    });
    expect(dialogMocks.updateTask).not.toHaveBeenCalled();

    submitForm({ title: "Implement refunds" });
    await waitFor(() => {
      expect(dialogMocks.updateTask).toHaveBeenCalledWith({
        id: "task-new",
        title: "Implement refunds",
        scope: "",
        assignee: "",
      });
    });
    expect(dialogMocks.createTask).toHaveBeenCalledOnce();
  });

  it("reports the created task and refreshes the graph when only the dependency fails", async () => {
    dialogMocks.addDependency.mockRejectedValueOnce(new Error("offline"));
    render(
      <CreateTaskDialog
        featureId="feature-1"
        onClose={vi.fn()}
        dependency={{ taskId: "task-1", direction: "blockedBy" }}
      />,
    );

    submitForm({ title: "Implement checkout" });

    await waitFor(() => {
      expect(
        screen.getByText(
          "The task was created, but adding the dependency failed. Submit again to retry only the dependency.",
        ),
      ).toBeInTheDocument();
    });
    expect(dialogMocks.refreshSnapshot).toHaveBeenCalledOnce();
  });

  it("fails the submission when creation returns no task id", async () => {
    dialogMocks.createTask.mockResolvedValue({});
    const onClose = vi.fn();
    render(
      <CreateTaskDialog
        featureId="feature-1"
        onClose={onClose}
        dependency={{ taskId: "task-1", direction: "blockedBy" }}
      />,
    );

    submitForm({ title: "Implement checkout" });

    await waitFor(() => {
      expect(dialogMocks.createTask).toHaveBeenCalledOnce();
    });
    expect(dialogMocks.addDependency).not.toHaveBeenCalled();
    expect(onClose).not.toHaveBeenCalled();
  });

  it("keeps the dialog open when creation fails and supports cancellation", async () => {
    dialogMocks.createTask.mockRejectedValueOnce(
      new Error("validation failed"),
    );
    const onClose = vi.fn();
    render(<CreateTaskDialog featureId="feature-1" onClose={onClose} />);

    submitForm({ title: "Implement checkout" });
    await waitFor(() => {
      expect(dialogMocks.createTask).toHaveBeenCalledOnce();
    });
    expect(onClose).not.toHaveBeenCalled();

    fireEvent.click(screen.getByRole("button", { name: "Cancel" }));
    expect(onClose).toHaveBeenCalledOnce();
  });

  it("disables submission while the mutation is pending and shows its error", () => {
    dialogMocks.mutation.isPending = true;
    dialogMocks.mutation.error = new Error("server rejected task");
    render(<CreateTaskDialog featureId="feature-1" onClose={vi.fn()} />);

    expect(screen.getByRole("button", { name: "Add task" })).toBeDisabled();
    expect(screen.getAllByRole("alert")).toHaveLength(1);
    expect(screen.getByRole("alert")).toHaveTextContent("server rejected task");
  });
});
