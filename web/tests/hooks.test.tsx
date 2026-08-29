import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { useDomainMutation, useSnapshot } from "../src/hooks";
import { makeSnapshot } from "./factories";

const hookMocks = vi.hoisted(() => ({ getSnapshot: vi.fn() }));

vi.mock("../src/api", () => ({ getSnapshot: hookMocks.getSnapshot }));

function createWrapper(queryClient: QueryClient) {
  return function Wrapper({ children }: { children: React.ReactNode }) {
    return (
      <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
    );
  };
}

describe("domain query hooks", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("loads snapshots through the query client", async () => {
    const snapshot = makeSnapshot();
    hookMocks.getSnapshot.mockResolvedValue(snapshot);
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });

    const { result } = renderHook(() => useSnapshot(), {
      wrapper: createWrapper(queryClient),
    });
    await waitFor(() => {
      expect(result.current.data).toBe(snapshot);
    });
    expect(hookMocks.getSnapshot).toHaveBeenCalledOnce();
  });

  it("invalidates the snapshot after a successful domain mutation", async () => {
    const queryClient = new QueryClient({
      defaultOptions: { mutations: { retry: false } },
    });
    const invalidateQueries = vi.spyOn(queryClient, "invalidateQueries");
    const mutation = vi.fn().mockResolvedValue("done");
    const { result } = renderHook(() => useDomainMutation(mutation), {
      wrapper: createWrapper(queryClient),
    });

    await expect(result.current.mutateAsync("input")).resolves.toBe("done");
    expect(mutation.mock.calls[0]?.[0]).toBe("input");
    expect(invalidateQueries).toHaveBeenCalledWith({
      queryKey: ["snapshot"],
    });
  });
});
