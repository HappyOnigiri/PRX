import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { FeatureStatus, type Snapshot } from "../src/gen/prx/v1/prx_pb";
import { ActiveFeatures } from "../src/views/ActiveFeatures";
import { makeFeature, makeSnapshot } from "./factories";

const activeMocks = vi.hoisted(() => ({
  state: {
    data: undefined as Snapshot | undefined,
    isPending: true,
    error: null as Error | null,
    refetch: vi.fn(),
  },
}));

vi.mock("@tanstack/react-router", () => ({
  Link: ({ children }: { children: ReactNode }) => (
    <a href="#test">{children}</a>
  ),
}));
vi.mock("../src/hooks", () => ({
  useSnapshot: () => activeMocks.state,
}));

describe("ActiveFeatures", () => {
  afterEach(cleanup);
  beforeEach(() => {
    activeMocks.state.data = undefined;
    activeMocks.state.isPending = true;
    activeMocks.state.error = null;
    activeMocks.state.refetch.mockReset();
  });

  it("shows only the features still in flight", () => {
    activeMocks.state.isPending = false;
    activeMocks.state.data = makeSnapshot({
      features: [
        makeFeature({
          slug: "open-payments",
          title: "Open payments",
          taskCount: 5,
          mergedCount: 2,
        }),
        makeFeature({
          id: "completed",
          title: "Finished payments",
          displayStatus: FeatureStatus.COMPLETED,
        }),
        makeFeature({
          id: "archived",
          title: "Historical payments",
          archived: true,
        }),
      ],
    });
    render(<ActiveFeatures />);

    expect(
      screen.getByRole("heading", { name: "Active features" }),
    ).toBeInTheDocument();
    expect(screen.getByText("Open payments")).toBeInTheDocument();
    expect(screen.queryByText("Finished payments")).not.toBeInTheDocument();
    expect(screen.queryByText("Historical payments")).not.toBeInTheDocument();
    expect(screen.getByText("2/5 merged")).toBeInTheDocument();
    expect(screen.getByText("Active")).toBeInTheDocument();
  });

  it("shows a dedicated empty state", () => {
    activeMocks.state.isPending = false;
    activeMocks.state.data = makeSnapshot({ features: [] });
    render(<ActiveFeatures />);
    expect(
      screen.getByRole("heading", { name: "Nothing is in flight" }),
    ).toBeInTheDocument();
  });

  it("reports its own loading and error states with a retry", () => {
    const { rerender } = render(<ActiveFeatures />);
    expect(
      screen.getByRole("heading", { name: "Loading the working set…" }),
    ).toBeInTheDocument();

    activeMocks.state.isPending = false;
    activeMocks.state.error = new Error("database unavailable");
    rerender(<ActiveFeatures />);
    expect(
      screen.getByRole("heading", {
        name: "The working set could not be loaded",
      }),
    ).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Try again" }));
    expect(activeMocks.state.refetch).toHaveBeenCalledOnce();
  });
});
