import { createMemoryHistory, RouterProvider } from "@tanstack/react-router";
import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { router } from "../src/router";
import {
  makeFeature,
  makeProject,
  makePullRequest,
  makeSnapshot,
  makeTask,
} from "./factories";

const snapshot = makeSnapshot({
  projects: [makeProject()],
  features: [
    makeFeature({
      projectId: "project-1",
      taskCount: 3,
      readyCount: 1,
      reviewWaitingCount: 1,
      conflictCount: 1,
    }),
  ],
  tasks: [],
  pullRequests: [makePullRequest()],
  readyTasks: [
    makeTask(),
    makeTask({ id: "task-4", title: "Unowned work", assignee: "" }),
  ],
  reviewWaitingTasks: [makeTask({ id: "task-2" })],
  conflictTasks: [makeTask({ id: "task-3" })],
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
    mutateAsync: vi.fn(),
    isPending: false,
    error: null,
  }),
}));

describe("Dashboard", () => {
  afterEach(cleanup);
  beforeEach(async () => {
    localStorage.clear();
    router.update({ history: createMemoryHistory({ initialEntries: ["/"] }) });
    await router.load();
  });

  it("shows derived queues and the next task", async () => {
    render(<RouterProvider router={router} />);
    expect(
      await screen.findByRole("heading", { name: /What can move/ }),
    ).toBeInTheDocument();
    expect(
      screen.queryByText(
        "Every queue is derived from the dependency graph—never manually marked ready.",
      ),
    ).not.toBeInTheDocument();
    expect(screen.queryByText("nodes under control")).not.toBeInTheDocument();
    expect(screen.getByText("Build API")).toBeInTheDocument();
    // キューの task は所属プロジェクトを示し、どの値もフィールド名を伴って
    // 表示される。
    expect(
      screen.getAllByText("Project")[0]?.nextElementSibling,
    ).toHaveTextContent("Delivery platform");
    expect(
      screen.getAllByText("Feature")[0]?.nextElementSibling,
    ).toHaveTextContent("Payments rollout");
    // 2 つ目の task には担当者がいないため、担当者の組は担当ありの task だけ
    // が持つ。
    expect(screen.getByText("Assignee").nextElementSibling).toHaveTextContent(
      "Bob",
    );
    expect(screen.queryByText("Unassigned")).not.toBeInTheDocument();
    expect(screen.getAllByText("not started")).toHaveLength(2);
    expect(screen.getByText("Conflicts")).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: "Sync GitHub" }),
    ).toBeInTheDocument();
    // pull request がある task は直接リンクするため、feature を開かずに
    // レビューへ辿り着ける。
    const pullRequest = screen.getByRole("link", { name: /acme\/prx #42/ });
    expect(pullRequest).toHaveAttribute(
      "href",
      "https://github.com/acme/prx/pull/42",
    );
    // 次の task を選ぶのは ready ボードなので、agent に渡すプロンプトは
    // 1 画面奥ではなくここでコピーする。
    expect(
      screen.getAllByRole("button", { name: "Copy design prompt" }),
    ).toHaveLength(2);
  });
});
