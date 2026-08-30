import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { Snapshot } from "../src/gen/prx/v1/prx_pb";
import { ArchivedFeatures } from "../src/views/ArchivedFeatures";
import { makeFeature, makeSnapshot } from "./factories";

const archivedMocks = vi.hoisted(() => ({
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
  useSnapshot: () => archivedMocks.state,
}));

describe("ArchivedFeatures", () => {
  afterEach(cleanup);
  beforeEach(() => {
    archivedMocks.state.data = undefined;
    archivedMocks.state.isPending = true;
    archivedMocks.state.error = null;
    archivedMocks.state.refetch.mockReset();
  });

  it("renders loading, error, and retry states", () => {
    const { rerender } = render(<ArchivedFeatures />);
    expect(
      screen.getByRole("heading", { name: "Loading archived features…" }),
    ).toBeInTheDocument();

    archivedMocks.state.isPending = false;
    archivedMocks.state.error = new Error("database unavailable");
    rerender(<ArchivedFeatures />);
    expect(screen.getByText("database unavailable")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Try again" }));
    expect(archivedMocks.state.refetch).toHaveBeenCalledOnce();
  });

  it("shows only archived features with progress and status", () => {
    archivedMocks.state.isPending = false;
    archivedMocks.state.data = makeSnapshot({
      features: [
        makeFeature({ title: "Active graph" }),
        makeFeature({
          id: "archived",
          slug: "historical-payments",
          title: "Historical payments",
          archived: true,
          taskCount: 5,
          mergedCount: 3,
        }),
      ],
    });
    render(<ArchivedFeatures />);

    expect(screen.queryByText("Active graph")).not.toBeInTheDocument();
    expect(screen.getByText("Historical payments")).toBeInTheDocument();
    expect(screen.getByText("3/5 merged")).toBeInTheDocument();
    expect(screen.getByText("Active")).toBeInTheDocument();
  });

  it("shows a dedicated empty state", () => {
    archivedMocks.state.isPending = false;
    archivedMocks.state.data = makeSnapshot({ features: [] });
    render(<ArchivedFeatures />);
    expect(
      screen.getByRole("heading", { name: "No archived features" }),
    ).toBeInTheDocument();
  });
});
