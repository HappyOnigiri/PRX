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
import { makeFeature } from "./factories";

const dialogMocks = vi.hoisted(() => ({
  mutation: {
    mutateAsync: vi.fn(),
    isPending: false,
    error: null as Error | null,
  },
}));

vi.mock("../src/api", () => ({ mutations: { updateFeature: vi.fn() } }));
vi.mock("../src/hooks", () => ({
  useDomainMutation: () => dialogMocks.mutation,
}));

describe("EditFeatureDialog", () => {
  afterEach(cleanup);
  beforeEach(() => {
    dialogMocks.mutation.mutateAsync.mockReset();
    dialogMocks.mutation.mutateAsync.mockResolvedValue({});
    dialogMocks.mutation.isPending = false;
    dialogMocks.mutation.error = null;
  });

  it("submits edited feature fields and closes on success", async () => {
    const onClose = vi.fn();
    render(
      <EditFeatureDialog
        feature={makeFeature({
          slug: "payments",
          title: "Payments",
          description: "Initial scope",
          status: FeatureStatus.ACTIVE,
        })}
        onClose={onClose}
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
    fireEvent.change(screen.getByRole("combobox"), {
      target: { value: String(FeatureStatus.PAUSED) },
    });
    fireEvent.submit(screen.getByRole("form", { name: "Edit feature" }));

    await waitFor(() => {
      expect(onClose).toHaveBeenCalledOnce();
    });
    expect(dialogMocks.mutation.mutateAsync).toHaveBeenCalledWith({
      id: "feature-1",
      slug: "payments-v2",
      title: "Payments v2",
      description: "Updated scope",
      status: FeatureStatus.PAUSED,
    });
  });

  it("keeps the dialog open after a failed update and allows cancellation", async () => {
    dialogMocks.mutation.mutateAsync.mockRejectedValueOnce(
      new Error("feature update failed"),
    );
    const onClose = vi.fn();
    render(<EditFeatureDialog feature={makeFeature()} onClose={onClose} />);

    fireEvent.submit(screen.getByRole("form", { name: "Edit feature" }));
    await waitFor(() => {
      expect(dialogMocks.mutation.mutateAsync).toHaveBeenCalled();
    });
    expect(onClose).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole("button", { name: "Cancel" }));
    expect(onClose).toHaveBeenCalledOnce();
  });

  it("disables save while a feature update is pending", () => {
    dialogMocks.mutation.isPending = true;
    render(<EditFeatureDialog feature={makeFeature()} onClose={vi.fn()} />);
    expect(screen.getByRole("button", { name: "Save feature" })).toBeDisabled();
  });
});
