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
  addDependency: vi.fn(),
  // ダイアログが useDomainMutation に渡した本体。mutateAsync 経由で実行して、
  // 作成と依存追加の並びを観察する。
  mutationFn: undefined as ((input: unknown) => Promise<unknown>) | undefined,
}));

vi.mock("../src/api", () => ({
  mutations: {
    createTask: dialogMocks.createTask,
    addDependency: dialogMocks.addDependency,
  },
}));
vi.mock("../src/hooks", () => ({
  useDomainMutation: (mutationFn: (input: unknown) => Promise<unknown>) => {
    dialogMocks.mutationFn = mutationFn;
    return dialogMocks.mutation;
  },
}));

describe("CreateTaskDialog", () => {
  afterEach(cleanup);
  beforeEach(() => {
    dialogMocks.mutation.mutateAsync.mockReset();
    dialogMocks.mutation.mutateAsync.mockResolvedValue({});
    dialogMocks.mutation.isPending = false;
    dialogMocks.mutation.error = null;
    dialogMocks.createTask.mockReset();
    dialogMocks.createTask.mockResolvedValue({ task: { id: "task-new" } });
    dialogMocks.addDependency.mockReset();
    dialogMocks.addDependency.mockResolvedValue({});
    dialogMocks.mutationFn = undefined;
  });

  function runMutation(input: {
    featureId: string;
    title: string;
    scope: string;
    assignee: string;
  }) {
    if (!dialogMocks.mutationFn) throw new Error("mutation body missing");
    return dialogMocks.mutationFn(input);
  }

  const taskInput = {
    featureId: "feature-1",
    title: "Implement checkout",
    scope: "",
    assignee: "",
  };

  it("converts form values into a task mutation and closes on success", async () => {
    const onClose = vi.fn();
    render(<CreateTaskDialog featureId="feature-1" onClose={onClose} />);
    fireEvent.change(screen.getByLabelText("Title"), {
      target: { value: "Implement checkout" },
    });
    fireEvent.change(screen.getByLabelText("Scope"), {
      target: { value: "API and acceptance tests" },
    });
    fireEvent.change(screen.getByLabelText("Assignee"), {
      target: { value: "Carol" },
    });

    fireEvent.submit(screen.getByRole("form", { name: "Create task" }));
    await waitFor(() => {
      expect(onClose).toHaveBeenCalledOnce();
    });
    expect(dialogMocks.mutation.mutateAsync).toHaveBeenCalledWith({
      featureId: "feature-1",
      title: "Implement checkout",
      scope: "API and acceptance tests",
      assignee: "Carol",
    });
  });

  it("adds the dependency after creating the task", async () => {
    render(
      <CreateTaskDialog
        featureId="feature-1"
        onClose={vi.fn()}
        dependency={{ taskId: "task-1", direction: "blockedBy" }}
        dependencyTitle="Blocker task"
      />,
    );
    expect(
      screen.getByText("The new task will be blocked by Blocker task."),
    ).toBeInTheDocument();

    await runMutation(taskInput);

    expect(dialogMocks.createTask).toHaveBeenCalledWith(taskInput);
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

    await runMutation(taskInput);

    expect(dialogMocks.addDependency).toHaveBeenCalledWith(
      "task-new",
      "task-1",
    );
  });

  it("retries only the dependency when the task already exists", async () => {
    dialogMocks.addDependency.mockRejectedValueOnce(new Error("offline"));
    render(
      <CreateTaskDialog
        featureId="feature-1"
        onClose={vi.fn()}
        dependency={{ taskId: "task-1", direction: "blockedBy" }}
      />,
    );

    await expect(runMutation(taskInput)).rejects.toThrow("offline");
    await runMutation(taskInput);

    expect(dialogMocks.createTask).toHaveBeenCalledOnce();
    expect(dialogMocks.addDependency).toHaveBeenCalledTimes(2);
  });

  it("keeps the dialog open when creation fails and supports cancellation", async () => {
    dialogMocks.mutation.mutateAsync.mockRejectedValueOnce(
      new Error("validation failed"),
    );
    const onClose = vi.fn();
    render(<CreateTaskDialog featureId="feature-1" onClose={onClose} />);

    fireEvent.submit(screen.getByRole("form", { name: "Create task" }));
    await waitFor(() => {
      expect(dialogMocks.mutation.mutateAsync).toHaveBeenCalled();
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
