import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { FeatureCreateDialog } from "../src/views/FeatureCreateDialog";
import { makeFeature } from "./factories";

const dialogMocks = vi.hoisted(() => ({
  navigate: vi.fn().mockResolvedValue(undefined),
  mutation: {
    mutateAsync: vi.fn(),
    isPending: false,
    error: null as Error | null,
  },
}));

vi.mock("@tanstack/react-router", () => ({
  useNavigate: () => dialogMocks.navigate,
}));
vi.mock("../src/api", () => ({
  mutations: { createFeature: vi.fn() },
}));
vi.mock("../src/hooks", () => ({
  useDomainMutation: () => dialogMocks.mutation,
}));

describe("FeatureCreateDialog", () => {
  afterEach(cleanup);
  beforeEach(() => {
    dialogMocks.navigate.mockClear();
    dialogMocks.mutation.mutateAsync.mockReset();
    dialogMocks.mutation.mutateAsync.mockResolvedValue({
      feature: makeFeature({ id: "F-9" }),
    });
    dialogMocks.mutation.isPending = false;
    dialogMocks.mutation.error = null;
  });

  // The dialog is opened from a project, so the membership comes from the page
  // rather than from a field the caller has to fill in.
  it("creates the feature in the project it was opened from", async () => {
    const onClose = vi.fn();
    render(<FeatureCreateDialog projectId="P-1" onClose={onClose} />);

    fireEvent.change(screen.getByLabelText("Title"), {
      target: { value: "Release" },
    });
    fireEvent.change(screen.getByLabelText("Description"), {
      target: { value: "Ship it" },
    });
    fireEvent.submit(screen.getByRole("form", { name: "Create feature" }));

    await waitFor(() => {
      expect(dialogMocks.navigate).toHaveBeenCalledOnce();
    });
    expect(dialogMocks.mutation.mutateAsync).toHaveBeenCalledWith({
      title: "Release",
      description: "Ship it",
      projectId: "P-1",
    });
    expect(dialogMocks.navigate).toHaveBeenCalledWith({
      to: "/features/$featureId",
      params: { featureId: "F-9" },
    });
    expect(onClose).toHaveBeenCalledOnce();
  });

  // A refused write leaves the dialog open with the reason on screen, so the
  // caller can correct the form instead of losing what they typed.
  it("keeps the dialog open and reports a refused creation", async () => {
    const onClose = vi.fn();
    dialogMocks.mutation.mutateAsync.mockRejectedValue(new Error("refused"));
    render(<FeatureCreateDialog projectId="P-1" onClose={onClose} />);

    fireEvent.change(screen.getByLabelText("Title"), {
      target: { value: "Release" },
    });
    fireEvent.submit(screen.getByRole("form", { name: "Create feature" }));

    await waitFor(() => {
      expect(dialogMocks.mutation.mutateAsync).toHaveBeenCalledOnce();
    });
    expect(onClose).not.toHaveBeenCalled();
    expect(dialogMocks.navigate).not.toHaveBeenCalled();
  });

  it("closes without creating anything on cancel", () => {
    const onClose = vi.fn();
    render(<FeatureCreateDialog projectId="P-1" onClose={onClose} />);

    fireEvent.click(screen.getByRole("button", { name: "Cancel" }));
    expect(onClose).toHaveBeenCalledOnce();
    expect(dialogMocks.mutation.mutateAsync).not.toHaveBeenCalled();
  });
});
