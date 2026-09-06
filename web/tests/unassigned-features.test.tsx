import { createMemoryHistory, RouterProvider } from "@tanstack/react-router";
import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { FeatureStatus } from "../src/gen/prx/v1/prx_pb";
import { router } from "../src/router";
import { makeFeature, makeProject, makeSnapshot } from "./factories";

const snapshot = makeSnapshot({
  projects: [makeProject({ id: "P-1", title: "Delivery platform" })],
  features: [
    makeFeature({ id: "F-1", title: "Loose end" }),
    makeFeature({
      id: "F-2",
      title: "Finished errand",
      displayStatus: FeatureStatus.COMPLETED,
      readyCount: 0,
    }),
    makeFeature({ id: "F-3", title: "Owned work", projectId: "P-1" }),
  ],
  tasks: [],
});

vi.mock("../src/hooks", () => ({
  useSnapshot: () => ({
    data: snapshot,
    isPending: false,
    isError: false,
    error: null,
    refetch: vi.fn(),
  }),
  useAutoSync: () => ({
    status: { data: undefined, isError: false },
    checking: false,
    error: null,
  }),
  useDomainMutation: () => ({
    mutate: vi.fn(),
    mutateAsync: vi.fn().mockResolvedValue({}),
    isPending: false,
    error: null,
  }),
}));

// The router is a module singleton whose load only settles once per file, so
// every step runs inside one mounted tree instead of remounting per case.
describe("Unassigned features route", () => {
  afterEach(cleanup);
  beforeEach(async () => {
    localStorage.clear();
    router.update({
      history: createMemoryHistory({
        initialEntries: ["/projects/unassigned"],
      }),
    });
    await router.load();
  });

  // "unassigned" is a static segment, which the router has to rank above the
  // dynamic project segment. Losing that would render the project workspace
  // with "unassigned" as a project ID and show "Project not found".
  it("opens its own page and keeps the chosen tab in the URL", async () => {
    render(<RouterProvider router={router} />);

    expect(
      await screen.findByRole("heading", { name: "No project", level: 1 }),
    ).toBeInTheDocument();
    expect(
      screen.queryByRole("heading", { name: "Project not found" }),
    ).not.toBeInTheDocument();

    // Only the open panel is in the accessibility tree; the folded ones keep
    // their rows in the DOM, so the assertions read the open one.
    const features = screen.getByRole("tabpanel");
    expect(features).toHaveTextContent("Loose end");
    expect(features).not.toHaveTextContent("Owned work");
    expect(features).not.toHaveTextContent("Finished errand");

    // The tab travels through the URL, so reload, history, and a shared link
    // reproduce the view.
    fireEvent.click(screen.getByRole("tab", { name: "Completed" }));
    await waitFor(() => {
      expect(router.state.location.search).toEqual({ features: "completed" });
    });
    expect(screen.getByRole("tabpanel")).toHaveTextContent("Finished errand");
  });
});
