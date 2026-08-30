import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { ImplementationPlanSection } from "../src/views/TaskInspectorImplementationPlan";

const planMocks = vi.hoisted(() => ({
  api: {
    getImplementationPlan: vi.fn(),
    upsertImplementationPlan: vi.fn(),
    deleteImplementationPlan: vi.fn(),
  },
  hookIndex: 0,
  mutations: Array.from({ length: 2 }, () => ({
    mutateAsync: vi.fn().mockResolvedValue({}),
    isPending: false,
    error: null as Error | null,
  })),
}));

vi.mock("../src/api", () => ({ mutations: planMocks.api }));
vi.mock("../src/hooks", () => ({
  useDomainMutation: () => {
    const mutation =
      planMocks.mutations[planMocks.hookIndex++ % planMocks.mutations.length];
    if (!mutation) throw new Error("mutation mock missing");
    return mutation;
  },
}));

function mutationAt(index: number) {
  const mutation = planMocks.mutations[index];
  if (!mutation) throw new Error(`mutation mock missing at ${index}`);
  return mutation;
}

describe("ImplementationPlanSection", () => {
  afterEach(() => {
    cleanup();
    vi.restoreAllMocks();
  });

  beforeEach(() => {
    planMocks.hookIndex = 0;
    planMocks.api.getImplementationPlan.mockReset();
    planMocks.api.upsertImplementationPlan.mockReset();
    planMocks.api.deleteImplementationPlan.mockReset();
    for (const mutation of planMocks.mutations) {
      mutation.mutateAsync.mockReset();
      mutation.mutateAsync.mockResolvedValue({});
      mutation.isPending = false;
      mutation.error = null;
    }
  });

  it("loads, saves, and deletes a Markdown plan", async () => {
    planMocks.api.getImplementationPlan.mockResolvedValue({
      implementationPlan: { content: "# Existing plan" },
    });
    mutationAt(0).mutateAsync.mockResolvedValue({
      implementationPlan: { content: "# Updated plan" },
    });
    render(<ImplementationPlanSection taskId="task-1" hasPlan />);

    await waitFor(() => {
      expect(screen.getByDisplayValue("# Existing plan")).toBeInTheDocument();
    });
    expect(planMocks.api.getImplementationPlan).toHaveBeenCalledWith("task-1");
    fireEvent.change(screen.getByRole("textbox"), {
      target: { value: "# Updated plan" },
    });
    fireEvent.click(screen.getByRole("button", { name: "Save plan" }));
    await waitFor(() => {
      expect(mutationAt(0).mutateAsync).toHaveBeenCalledWith({
        taskId: "task-1",
        content: "# Updated plan",
      });
    });
    fireEvent.click(screen.getByRole("button", { name: "Delete plan" }));
    await waitFor(() => {
      expect(mutationAt(1).mutateAsync).toHaveBeenCalledWith("task-1");
    });
  });

  it("shows an empty editor without fetching an unregistered plan", () => {
    render(<ImplementationPlanSection taskId="task-1" hasPlan={false} />);

    expect(
      screen.getByText("No implementation plan is registered."),
    ).toBeInTheDocument();
    expect(planMocks.api.getImplementationPlan).not.toHaveBeenCalled();
    expect(screen.getByRole("textbox")).toHaveValue("");
  });

  it("shows a read error when the registered plan cannot be loaded", async () => {
    planMocks.api.getImplementationPlan.mockRejectedValueOnce(
      new Error("plan file unavailable"),
    );
    render(<ImplementationPlanSection taskId="task-1" hasPlan />);

    expect(await screen.findByRole("alert")).toHaveTextContent(
      "plan file unavailable",
    );
    expect(screen.getByRole("textbox")).toHaveValue("");
  });
});
