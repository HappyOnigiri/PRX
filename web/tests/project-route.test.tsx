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
    makeProject({ id: "P-1", title: "Delivery platform" }),
    makeProject({
      id: "P-2",
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

  // アーカイブ表示は URL の状態なのでリロード・履歴・共有リンクで再現でき、
  // サイドバーからはどの画面幅でもこのページへ辿り着ける。
  it("keeps the archive tab in the URL and links from the sidebar", async () => {
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
    // 件数は複数形キーなので、feature が 1 つのプロジェクトが
    // "1 features" と表示されてはいけない。
    expect(list).toHaveTextContent("1 feature");
    expect(list).not.toHaveTextContent("1 features");

    fireEvent.click(screen.getByRole("tab", { name: "Archived" }));
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
