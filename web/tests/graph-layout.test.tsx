import { createMemoryHistory, RouterProvider } from "@tanstack/react-router";
import { render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { router } from "../src/router";
import { makeSnapshot } from "./factories";

// React Flow measures its container, which jsdom does not implement.
class ResizeObserverStub {
  observe() {}
  unobserve() {}
  disconnect() {}
}
globalThis.ResizeObserver = ResizeObserverStub;

const snapshot = makeSnapshot();

vi.mock("../src/hooks", () => ({
  useSnapshot: () => ({
    data: snapshot,
    isPending: false,
    isError: false,
    error: null,
    refetch: vi.fn(),
  }),
  useDomainMutation: () => ({
    mutate: vi.fn(),
    mutateAsync: vi.fn(),
    isPending: false,
    error: null,
  }),
}));

vi.mock("elkjs/lib/elk-api.js", () => ({
  default: class {
    layout() {
      return Promise.reject(new Error("worker unavailable"));
    }
    terminateWorker() {}
  },
}));

describe("FeatureWorkspace graph layout", () => {
  beforeEach(async () => {
    router.update({
      history: createMemoryHistory({ initialEntries: ["/features/feature-1"] }),
    });
    await router.load();
  });

  it("surfaces a layout failure instead of leaving the canvas blank", async () => {
    render(<RouterProvider router={router} />);
    const alert = await screen.findByRole("alert");
    expect(alert).toHaveTextContent("Graph layout failed");
    expect(alert).toHaveTextContent("worker unavailable");
    expect(
      screen.getByRole("button", { name: "Retry layout" }),
    ).toBeInTheDocument();
  });
});
