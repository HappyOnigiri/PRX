import { createMemoryHistory, RouterProvider } from "@tanstack/react-router";
import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { router } from "../src/router";
import { makeFeature, makeProject, makeSnapshot } from "./factories";

const snapshot = makeSnapshot({
  projects: [
    makeProject({ id: "P-1", slug: "delivery", title: "Delivery platform" }),
    makeProject({
      id: "P-2",
      slug: "sunset",
      title: "Sunset initiative",
      archived: true,
    }),
  ],
  features: [makeFeature({ id: "F-1", title: "Checkout", projectId: "P-1" })],
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

describe("Project list route", () => {
  afterEach(cleanup);

  beforeEach(async () => {
    localStorage.clear();
    router.update({
      history: createMemoryHistory({ initialEntries: ["/projects"] }),
    });
    await router.load();
  });

  // The archived view is a URL state so reload, history, and a shared link
  // reproduce it, and the sidebar reaches the page at every viewport width.
  it("keeps the archive toggle in the URL and links from the sidebar", async () => {
    render(<RouterProvider router={router} />);

    const navigation = screen.getByRole("navigation", {
      name: "PRX navigation",
    });
    expect(navigation).toHaveTextContent("Projects");
    expect(navigation).toHaveTextContent("Delivery platform");
    expect(navigation).not.toHaveTextContent("Sunset initiative");

    const list = screen.getByRole("region", { name: "Project list" });
    expect(list).toHaveTextContent("Delivery platform");
    expect(list).not.toHaveTextContent("Sunset initiative");
    // The count is a plural key, so a project with one feature must not read
    // "1 features".
    expect(list).toHaveTextContent("1 feature");
    expect(list).not.toHaveTextContent("1 features");

    fireEvent.click(screen.getByRole("button", { name: "Show archived" }));
    await waitFor(() => {
      expect(router.state.location.search).toEqual({ archived: true });
    });
    expect(
      screen.getByRole("region", { name: "Project list" }),
    ).toHaveTextContent("Sunset initiative");

    router.history.back();
    await waitFor(() => {
      expect(router.state.location.search).toEqual({ archived: false });
    });
  });
});
