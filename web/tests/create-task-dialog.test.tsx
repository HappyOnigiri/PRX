import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { TaskKind } from "../src/gen/prx/v1/prx_pb";
import { CreateTaskDialog } from "../src/views/CreateTaskDialog";

const dialogMocks = vi.hoisted(() => ({
  mutation: {
    mutateAsync: vi.fn(),
    isPending: false,
    error: null as Error | null,
  },
}));

vi.mock("../src/api", () => ({ mutations: { createTask: vi.fn() } }));
vi.mock("../src/hooks", () => ({
  useDomainMutation: () => dialogMocks.mutation,
}));

describe("CreateTaskDialog", () => {
  afterEach(cleanup);
  beforeEach(() => {
    dialogMocks.mutation.mutateAsync.mockReset();
    dialogMocks.mutation.mutateAsync.mockResolvedValue({});
    dialogMocks.mutation.isPending = false;
    dialogMocks.mutation.error = null;
  });

  it("converts form values into a task mutation and closes on success", async () => {
    const onClose = vi.fn();
    render(<CreateTaskDialog featureId="feature-1" onClose={onClose} />);
    fireEvent.change(screen.getByLabelText("Title"), {
      target: { value: "Implement checkout" },
    });
    fireEvent.change(screen.getByLabelText("Scope"), {
      target: { value: "API and acceptance tests" },
    });
    fireEvent.change(screen.getByRole("combobox"), {
      target: { value: String(TaskKind.MANUAL) },
    });
    fireEvent.change(screen.getByLabelText("Assignee"), {
      target: { value: "Ren" },
    });

    fireEvent.submit(screen.getByRole("form", { name: "Create task" }));
    await waitFor(() => {
      expect(onClose).toHaveBeenCalledOnce();
    });
    expect(dialogMocks.mutation.mutateAsync).toHaveBeenCalledWith({
      featureId: "feature-1",
      title: "Implement checkout",
      scope: "API and acceptance tests",
      kind: TaskKind.MANUAL,
      assignee: "Ren",
    });
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
