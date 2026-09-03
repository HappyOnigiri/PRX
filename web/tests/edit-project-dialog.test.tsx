import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { EditProjectDialog } from "../src/views/EditProjectDialog";
import { makeProject } from "./factories";

const dialogMocks = vi.hoisted(() => ({
  hookIndex: 0,
  mutations: Array.from({ length: 2 }, () => ({
    mutateAsync: vi.fn(),
    isPending: false,
    error: null as Error | null,
  })),
}));

vi.mock("../src/api", () => ({
  mutations: { updateProject: vi.fn(), deleteProject: vi.fn() },
}));
vi.mock("../src/hooks", () => ({
  useDomainMutation: () => {
    const mutation = dialogMocks.mutations[dialogMocks.hookIndex++ % 2];
    if (!mutation) throw new Error("mutation mock missing");
    return mutation;
  },
}));

function mutationAt(index: number) {
  const mutation = dialogMocks.mutations[index];
  if (!mutation) throw new Error(`mutation mock missing at ${index}`);
  return mutation;
}

describe("EditProjectDialog", () => {
  afterEach(cleanup);
  beforeEach(() => {
    dialogMocks.hookIndex = 0;
    for (const mutation of dialogMocks.mutations) {
      mutation.mutateAsync.mockReset();
      mutation.mutateAsync.mockResolvedValue({});
      mutation.isPending = false;
      mutation.error = null;
    }
  });

  it("submits the editable fields of an active project", async () => {
    const onClose = vi.fn();
    render(
      <EditProjectDialog
        project={makeProject({
          id: "P-1",
          slug: "delivery",
          title: "Delivery",
        })}
        referenceCount={2}
        onClose={onClose}
        onDeleted={vi.fn()}
      />,
    );
    fireEvent.change(screen.getByLabelText("Slug"), {
      target: { value: "delivery-platform" },
    });
    fireEvent.change(screen.getByLabelText("Title"), {
      target: { value: "Delivery platform" },
    });
    fireEvent.change(screen.getByLabelText("Description"), {
      target: { value: "Shared work" },
    });
    fireEvent.submit(screen.getByRole("form", { name: "Edit project" }));

    await waitFor(() => {
      expect(onClose).toHaveBeenCalledOnce();
    });
    expect(mutationAt(0).mutateAsync).toHaveBeenCalledWith({
      id: "P-1",
      slug: "delivery-platform",
      title: "Delivery platform",
      description: "Shared work",
    });
  });

  it("confirms the archive and states that features are released on delete", async () => {
    const onClose = vi.fn();
    render(
      <EditProjectDialog
        project={makeProject({ id: "P-1", title: "Delivery" })}
        referenceCount={3}
        onClose={onClose}
        onDeleted={vi.fn()}
      />,
    );
    fireEvent.click(screen.getByRole("button", { name: "Archive project" }));
    expect(mutationAt(0).mutateAsync).not.toHaveBeenCalled();
    fireEvent.click(
      screen.getByRole("button", { name: "Archive project", hidden: false }),
    );
    await waitFor(() => {
      expect(mutationAt(0).mutateAsync).toHaveBeenCalledWith({
        id: "P-1",
        archived: true,
      });
    });
  });

  it("releases features when the delete is confirmed", async () => {
    const onDeleted = vi.fn();
    render(
      <EditProjectDialog
        project={makeProject({ id: "P-1", title: "Delivery" })}
        referenceCount={3}
        onClose={vi.fn()}
        onDeleted={onDeleted}
      />,
    );
    fireEvent.click(screen.getByRole("button", { name: "Delete project" }));
    expect(
      screen.getByRole("dialog", { name: "Delete Delivery?" }),
    ).toHaveTextContent("Its features are kept");
    fireEvent.click(screen.getByRole("button", { name: "Delete permanently" }));
    await waitFor(() => {
      expect(onDeleted).toHaveBeenCalledOnce();
    });
    expect(mutationAt(1).mutateAsync).toHaveBeenCalledWith("P-1");
  });

  it("keeps a failed delete open and cancels a confirmation", async () => {
    const onDeleted = vi.fn();
    mutationAt(1).mutateAsync.mockRejectedValue(new Error("delete failed"));
    mutationAt(1).error = new Error("delete failed");
    render(
      <EditProjectDialog
        project={makeProject({ id: "P-1", title: "Delivery" })}
        referenceCount={0}
        onClose={vi.fn()}
        onDeleted={onDeleted}
      />,
    );
    fireEvent.click(screen.getByRole("button", { name: "Delete project" }));
    fireEvent.click(screen.getByRole("button", { name: "Delete permanently" }));
    await waitFor(() => {
      expect(mutationAt(1).mutateAsync).toHaveBeenCalled();
    });
    expect(onDeleted).not.toHaveBeenCalled();
    expect(screen.getByText("delete failed")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Cancel" }));
    expect(
      screen.queryByRole("dialog", { name: "Delete Delivery?" }),
    ).not.toBeInTheDocument();
  });

  // An archived project shows its values without edit fields and offers the
  // activation that lifts the archive from every feature inside it.
  it("activates an archived project instead of editing it", async () => {
    const onClose = vi.fn();
    render(
      <EditProjectDialog
        project={makeProject({
          id: "P-1",
          title: "Delivery",
          description: "",
          archived: true,
        })}
        referenceCount={1}
        onClose={onClose}
        onDeleted={vi.fn()}
      />,
    );
    expect(
      screen.getByRole("dialog", { name: "Manage archived project" }),
    ).toBeInTheDocument();
    expect(screen.queryByLabelText("Slug")).not.toBeInTheDocument();
    expect(screen.getByText("No project description yet.")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Activate project" }));
    await waitFor(() => {
      expect(mutationAt(0).mutateAsync).toHaveBeenCalledWith({
        id: "P-1",
        archived: false,
      });
    });
    fireEvent.click(screen.getByRole("button", { name: "Close" }));
    expect(onClose).toHaveBeenCalled();
  });
});
