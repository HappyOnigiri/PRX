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
import { makeSnapshot, makeTask } from "./factories";

const snapshot = makeSnapshot({
  tasks: [makeTask({ title: "Release API" })],
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
}));

describe("Task search route", () => {
  afterEach(cleanup);

  beforeEach(async () => {
    localStorage.clear();
    router.update({
      history: createMemoryHistory({
        initialEntries: ["/tasks?q=task-status%3Aready"],
      }),
    });
    await router.load();
  });

  it("hydrates q, updates the URL, and follows browser history", async () => {
    render(<RouterProvider router={router} />);

    const input = screen.getByRole("textbox", {
      name: "Search active tasks",
    });
    expect(input).toHaveValue("task-status:ready");
    expect(screen.getByText("Release API")).toBeInTheDocument();

    fireEvent.change(input, { target: { value: "release" } });
    fireEvent.submit(screen.getByRole("search"));
    await waitFor(() => {
      expect(router.state.location.search).toEqual({ q: "release" });
      expect(screen.getByRole("textbox")).toHaveValue("release");
    });

    router.history.back();
    await waitFor(() => {
      expect(router.state.location.search).toEqual({
        q: "task-status:ready",
      });
      expect(screen.getByRole("textbox")).toHaveValue("task-status:ready");
    });
  });
});
