import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import {
  useConfig,
  useConfigMutation,
  useDomainMutation,
  useSnapshot,
} from "../src/hooks";
import { makeSnapshot } from "./factories";

const hookMocks = vi.hoisted(() => ({
  getSnapshot: vi.fn(),
  getConfig: vi.fn(),
  getSyncStatus: vi.fn(),
  syncIfDue: vi.fn(),
}));

vi.mock("../src/api", () => ({
  getSnapshot: hookMocks.getSnapshot,
  getConfig: hookMocks.getConfig,
  getSyncStatus: hookMocks.getSyncStatus,
  syncIfDue: hookMocks.syncIfDue,
}));

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

  it("checks automatic sync and invalidates the snapshot only after a run", async () => {
    const snapshot = makeSnapshot();
    const status = { intervalSeconds: 3600n, succeeded: 1, failed: 0 };
    hookMocks.getSnapshot.mockResolvedValue(snapshot);
    hookMocks.getSyncStatus.mockResolvedValue(status);
    hookMocks.syncIfDue.mockResolvedValue({ ran: true, status });
    const queryClient = new QueryClient({
      defaultOptions: {
        queries: { retry: false },
        mutations: { retry: false },
      },
    });
    const invalidateQueries = vi.spyOn(queryClient, "invalidateQueries");
    const { result, unmount } = renderHook(() => useSnapshot(true), {
      wrapper: createWrapper(queryClient),
    });
    await waitFor(() => {
      expect(hookMocks.syncIfDue).toHaveBeenCalled();
      expect(result.current.autoSync?.status.data).toBe(status);
    });
    expect(invalidateQueries).toHaveBeenCalledWith({ queryKey: ["snapshot"] });
    window.dispatchEvent(new Event("focus"));
    await waitFor(() => {
      expect(hookMocks.syncIfDue.mock.calls.length).toBeGreaterThan(1);
    });
    unmount();
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

  it("loads and invalidates the server configuration", async () => {
    const config = { version: 1 };
    hookMocks.getConfig.mockResolvedValue(config);
    const queryClient = new QueryClient({
      defaultOptions: {
        queries: { retry: false },
        mutations: { retry: false },
      },
    });
    const invalidateQueries = vi.spyOn(queryClient, "invalidateQueries");
    const { result } = renderHook(() => useConfig(), {
      wrapper: createWrapper(queryClient),
    });
    await waitFor(() => {
      expect(result.current.data).toBe(config);
    });
    expect(hookMocks.getConfig).toHaveBeenCalledOnce();

    const mutation = vi.fn().mockResolvedValue("saved");
    const mutationHook = renderHook(() => useConfigMutation(mutation), {
      wrapper: createWrapper(queryClient),
    });
    await expect(
      mutationHook.result.current.mutateAsync("input"),
    ).resolves.toBe("saved");
    expect(invalidateQueries).toHaveBeenCalledWith({
      queryKey: ["github-config"],
    });
  });
});
