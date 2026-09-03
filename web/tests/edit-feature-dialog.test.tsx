import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { FeatureStatus } from "../src/gen/prx/v1/prx_pb";
import { EditFeatureDialog } from "../src/views/EditFeatureDialog";
import { makeFeature, makeProject } from "./factories";

const dialogMocks = vi.hoisted(() => ({
  hookIndex: 0,
  mutations: Array.from({ length: 2 }, () => ({
    mutateAsync: vi.fn(),
    isPending: false,
    error: null as Error | null,
  })),
}));

vi.mock("../src/api", () => ({
  mutations: { updateFeature: vi.fn(), deleteFeature: vi.fn() },
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

describe("EditFeatureDialog", () => {
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

  it("submits active feature fields and keeps a failed edit open", async () => {
    const onClose = vi.fn();
    const { rerender } = render(
      <EditFeatureDialog
        projects={[]}
        feature={makeFeature({
          slug: "payments",
          title: "Payments",
          description: "Initial scope",
          status: FeatureStatus.ACTIVE,
        })}
        onClose={onClose}
        onDeleted={vi.fn()}
      />,
    );
    fireEvent.change(screen.getByLabelText("Slug"), {
      target: { value: "payments-v2" },
    });
    fireEvent.change(screen.getByLabelText("Title"), {
      target: { value: "Payments v2" },
    });
    fireEvent.change(screen.getByLabelText("Description"), {
      target: { value: "Updated scope" },
    });
    fireEvent.change(screen.getByLabelText("Status"), {
      target: { value: String(FeatureStatus.PAUSED) },
    });
    fireEvent.submit(screen.getByRole("form", { name: "Edit feature" }));

    await waitFor(() => {
      expect(onClose).toHaveBeenCalledOnce();
    });
    expect(mutationAt(0).mutateAsync).toHaveBeenCalledWith({
      id: "feature-1",
      slug: "payments-v2",
      title: "Payments v2",
      description: "Updated scope",
      status: FeatureStatus.PAUSED,
      projectId: "",
    });

    onClose.mockClear();
    dialogMocks.hookIndex = 0;
    mutationAt(0).mutateAsync.mockRejectedValueOnce(new Error("update failed"));
    rerender(
      <EditFeatureDialog
        projects={[]}
        feature={makeFeature()}
        onClose={onClose}
        onDeleted={vi.fn()}
      />,
    );
    fireEvent.submit(screen.getByRole("form", { name: "Edit feature" }));
    await waitFor(() => {
      expect(mutationAt(0).mutateAsync).toHaveBeenCalled();
    });
    expect(onClose).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole("button", { name: "Cancel" }));
    expect(onClose).toHaveBeenCalledOnce();
  });

  it("confirms completing a feature whose tasks are unfinished", async () => {
    const onClose = vi.fn();
    render(
      <EditFeatureDialog
        projects={[]}
        feature={makeFeature({
          title: "Payments",
          status: FeatureStatus.AUTO,
          taskCount: 4,
          finishedCount: 1,
        })}
        onClose={onClose}
        onDeleted={vi.fn()}
      />,
    );
    expect(screen.getByLabelText("Status")).toHaveValue(
      String(FeatureStatus.AUTO),
    );
    expect(
      screen.getByRole("option", { name: "Automatic" }),
    ).toBeInTheDocument();

    fireEvent.change(screen.getByLabelText("Status"), {
      target: { value: String(FeatureStatus.COMPLETED) },
    });
    fireEvent.submit(screen.getByRole("form", { name: "Edit feature" }));
    expect(mutationAt(0).mutateAsync).not.toHaveBeenCalled();
    expect(
      screen.getByRole("dialog", { name: "Complete Payments?" }),
    ).toHaveTextContent("3 of its tasks are still unfinished");

    fireEvent.click(screen.getByRole("button", { name: "Cancel" }));
    expect(
      screen.queryByRole("dialog", { name: "Complete Payments?" }),
    ).not.toBeInTheDocument();
    expect(mutationAt(0).mutateAsync).not.toHaveBeenCalled();

    fireEvent.submit(screen.getByRole("form", { name: "Edit feature" }));
    fireEvent.click(screen.getByRole("button", { name: "Complete feature" }));
    await waitFor(() => {
      expect(onClose).toHaveBeenCalledOnce();
    });
    expect(mutationAt(0).mutateAsync).toHaveBeenCalledWith({
      id: "feature-1",
      slug: "payments",
      title: "Payments",
      description: "",
      status: FeatureStatus.COMPLETED,
      projectId: "",
    });
  });

  it("completes a fully finished feature without a confirmation", async () => {
    const onClose = vi.fn();
    render(
      <EditFeatureDialog
        projects={[]}
        feature={makeFeature({ taskCount: 2, finishedCount: 2 })}
        onClose={onClose}
        onDeleted={vi.fn()}
      />,
    );
    fireEvent.change(screen.getByLabelText("Status"), {
      target: { value: String(FeatureStatus.COMPLETED) },
    });
    fireEvent.submit(screen.getByRole("form", { name: "Edit feature" }));
    await waitFor(() => {
      expect(onClose).toHaveBeenCalledOnce();
    });
    expect(mutationAt(0).mutateAsync).toHaveBeenCalledWith({
      id: "feature-1",
      slug: "payments",
      title: "Payments rollout",
      description: "",
      status: FeatureStatus.COMPLETED,
      projectId: "",
    });
  });

  it("confirms archive in-app and supports cancellation, Escape, and failure", async () => {
    const onClose = vi.fn();
    mutationAt(0).error = new Error("archive failed");
    mutationAt(0).mutateAsync.mockRejectedValue(new Error("archive failed"));
    render(
      <EditFeatureDialog
        projects={[]}
        feature={makeFeature({ title: "Payments" })}
        onClose={onClose}
        onDeleted={vi.fn()}
      />,
    );

    fireEvent.click(screen.getByRole("button", { name: "Archive feature" }));
    expect(
      screen.getByRole("dialog", { name: "Archive Payments?" }),
    ).toBeInTheDocument();
    expect(screen.getByRole("alert")).toHaveTextContent("archive failed");
    const archiveDialog = screen.getByRole("dialog", {
      name: "Archive Payments?",
    });
    const cancel = screen.getByRole("button", { name: "Cancel" });
    const confirm = screen.getByRole("button", { name: "Archive feature" });
    expect(cancel).toHaveFocus();
    fireEvent.keyDown(archiveDialog, { key: "Tab", shiftKey: true });
    expect(confirm).toHaveFocus();
    fireEvent.keyDown(archiveDialog, { key: "Tab" });
    expect(cancel).toHaveFocus();
    fireEvent.keyDown(archiveDialog, { key: "Escape" });
    expect(
      screen.queryByRole("dialog", { name: "Archive Payments?" }),
    ).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Archive feature" }));
    fireEvent.click(screen.getByRole("button", { name: "Archive feature" }));
    await waitFor(() => {
      expect(mutationAt(0).mutateAsync).toHaveBeenCalledWith({
        id: "feature-1",
        archived: true,
      });
    });
    expect(onClose).not.toHaveBeenCalled();

    fireEvent.click(screen.getByRole("button", { name: "Cancel" }));
    expect(
      screen.queryByRole("dialog", { name: "Archive Payments?" }),
    ).not.toBeInTheDocument();

    mutationAt(0).mutateAsync.mockResolvedValueOnce({});
    fireEvent.click(screen.getByRole("button", { name: "Archive feature" }));
    fireEvent.click(screen.getByRole("button", { name: "Archive feature" }));
    await waitFor(() => {
      expect(onClose).toHaveBeenCalledOnce();
    });
  });

  it("restores an archived feature and exposes values without edit fields", async () => {
    const onClose = vi.fn();
    render(
      <EditFeatureDialog
        projects={[]}
        feature={makeFeature({
          archived: true,
          title: "Archived payments",
          description: "Historical scope",
        })}
        onClose={onClose}
        onDeleted={vi.fn()}
      />,
    );

    expect(
      screen.getByRole("dialog", { name: "Manage archived feature" }),
    ).toHaveTextContent("Historical scope");
    expect(screen.queryByRole("form")).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Delete feature" }));
    expect(
      screen.getByRole("dialog", { name: "Delete Archived payments?" }),
    ).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Cancel" }));
    mutationAt(0).mutateAsync.mockRejectedValueOnce(
      new Error("restore failed"),
    );
    fireEvent.click(screen.getByRole("button", { name: "Restore feature" }));
    await waitFor(() => {
      expect(mutationAt(0).mutateAsync).toHaveBeenCalledWith({
        id: "feature-1",
        archived: false,
      });
    });
    expect(onClose).not.toHaveBeenCalled();
    mutationAt(0).mutateAsync.mockResolvedValueOnce({});
    fireEvent.click(screen.getByRole("button", { name: "Restore feature" }));
    await waitFor(() => {
      expect(onClose).toHaveBeenCalledOnce();
    });
    expect(mutationAt(0).mutateAsync).toHaveBeenCalledWith({
      id: "feature-1",
      archived: false,
    });
  });

  // Restoring a feature whose project is archived leaves it read-only, so the
  // dialog points at the project instead of offering that restore.
  it("withholds the restore when the project is archived too", () => {
    render(
      <EditFeatureDialog
        projects={[makeProject({ id: "project-1", archived: true })]}
        feature={makeFeature({ archived: true, projectId: "project-1" })}
        onClose={vi.fn()}
        onDeleted={vi.fn()}
      />,
    );

    expect(
      screen.queryByRole("button", { name: "Restore feature" }),
    ).not.toBeInTheDocument();
    expect(
      screen.getByRole("dialog", { name: "Manage archived feature" }),
    ).toHaveTextContent(
      "Its project is archived, so the archive is lifted there.",
    );
    expect(
      screen.getByRole("button", { name: "Delete feature" }),
    ).toBeInTheDocument();
  });

  it("confirms permanent deletion and keeps pending actions locked", async () => {
    const onDeleted = vi.fn();
    render(
      <EditFeatureDialog
        projects={[]}
        feature={makeFeature({ title: "Payments", taskCount: 3 })}
        onClose={vi.fn()}
        onDeleted={onDeleted}
      />,
    );
    fireEvent.click(screen.getByRole("button", { name: "Delete feature" }));
    expect(screen.getByText(/3 tasks/)).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Cancel" }));
    expect(mutationAt(1).mutateAsync).not.toHaveBeenCalled();

    mutationAt(1).mutateAsync.mockRejectedValueOnce(new Error("delete failed"));
    fireEvent.click(screen.getByRole("button", { name: "Delete feature" }));
    fireEvent.click(screen.getByRole("button", { name: "Delete permanently" }));
    await waitFor(() => {
      expect(mutationAt(1).mutateAsync).toHaveBeenCalledWith("feature-1");
    });
    expect(onDeleted).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole("button", { name: "Cancel" }));

    mutationAt(1).mutateAsync.mockResolvedValueOnce({});
    fireEvent.click(screen.getByRole("button", { name: "Delete feature" }));
    fireEvent.click(screen.getByRole("button", { name: "Delete permanently" }));
    await waitFor(() => {
      expect(onDeleted).toHaveBeenCalledOnce();
    });
    expect(mutationAt(1).mutateAsync).toHaveBeenCalledWith("feature-1");

    cleanup();
    dialogMocks.hookIndex = 0;
    mutationAt(1).isPending = false;
    const pendingRender = render(
      <EditFeatureDialog
        projects={[]}
        feature={makeFeature()}
        onClose={vi.fn()}
        onDeleted={vi.fn()}
      />,
    );
    fireEvent.click(screen.getByRole("button", { name: "Delete feature" }));
    mutationAt(1).isPending = true;
    pendingRender.rerender(
      <EditFeatureDialog
        projects={[]}
        feature={makeFeature()}
        onClose={vi.fn()}
        onDeleted={vi.fn()}
      />,
    );
    expect(
      screen.getByRole("button", { name: "Delete permanently" }),
    ).toBeDisabled();
    expect(screen.getByRole("button", { name: "Cancel" })).toBeDisabled();
    const pendingDialog = screen.getByRole("dialog", {
      name: "Delete Payments rollout?",
    });
    fireEvent.keyDown(pendingDialog, { key: "Tab" });
    fireEvent.keyDown(pendingDialog, { key: "Escape" });
    expect(pendingDialog).toBeInTheDocument();
  });
});
