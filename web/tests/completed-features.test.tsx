import { cleanup, render, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { FeatureStatus, type Snapshot } from "../src/gen/prx/v1/prx_pb";
import { CompletedFeatures } from "../src/views/CompletedFeatures";
import { makeFeature, makeSnapshot } from "./factories";

const completedMocks = vi.hoisted(() => ({
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
  useSnapshot: () => completedMocks.state,
}));

describe("CompletedFeatures", () => {
  afterEach(cleanup);
  beforeEach(() => {
    completedMocks.state.data = undefined;
    completedMocks.state.isPending = true;
    completedMocks.state.error = null;
    completedMocks.state.refetch.mockReset();
  });

  it("shows only the completed features that are still in the working set", () => {
    completedMocks.state.isPending = false;
    completedMocks.state.data = makeSnapshot({
      features: [
        makeFeature({ title: "Active graph" }),
        makeFeature({
          id: "completed",
          slug: "finished-payments",
          title: "Finished payments",
          displayStatus: FeatureStatus.COMPLETED,
          taskCount: 5,
          mergedCount: 4,
          finishedCount: 5,
        }),
        makeFeature({
          id: "archived-and-completed",
          slug: "archived-payments",
          title: "Archived payments",
          archived: true,
          displayStatus: FeatureStatus.COMPLETED,
        }),
      ],
    });
    render(<CompletedFeatures />);

    expect(
      screen.getByRole("heading", { name: "Completed features" }),
    ).toBeInTheDocument();
    expect(screen.getByText("Finished payments")).toBeInTheDocument();
    expect(screen.queryByText("Active graph")).not.toBeInTheDocument();
    expect(screen.queryByText("Archived payments")).not.toBeInTheDocument();
    expect(screen.getByText("4/5 merged")).toBeInTheDocument();
    expect(screen.getByText("Completed")).toBeInTheDocument();
  });

  it("shows a dedicated empty state", () => {
    completedMocks.state.isPending = false;
    completedMocks.state.data = makeSnapshot({ features: [] });
    render(<CompletedFeatures />);
    expect(
      screen.getByRole("heading", { name: "No completed features" }),
    ).toBeInTheDocument();
  });

  it("reports its own loading and error states", () => {
    const { rerender } = render(<CompletedFeatures />);
    expect(
      screen.getByRole("heading", { name: "Loading completed features…" }),
    ).toBeInTheDocument();

    completedMocks.state.isPending = false;
    completedMocks.state.error = new Error("database unavailable");
    rerender(<CompletedFeatures />);
    expect(
      screen.getByRole("heading", {
        name: "Completed features could not be loaded",
      }),
    ).toBeInTheDocument();
  });
});
