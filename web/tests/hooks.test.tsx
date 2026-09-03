import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import {
  useAutoSync,
  useConfig,
  useConfigMutation,
  useDebugReport,
  useDomainMutation,
  useQueryDiagnostics,
  useSnapshot,
} from "../src/hooks";
import { makeSnapshot } from "./factories";

const hookMocks = vi.hoisted(() => ({
  getSnapshot: vi.fn(),
  getDebugReport: vi.fn(),
  getConfig: vi.fn(),
  getSyncStatus: vi.fn(),
  syncIfDue: vi.fn(),
}));

vi.mock("../src/api", () => ({
  getSnapshot: hookMocks.getSnapshot,
  getDebugReport: hookMocks.getDebugReport,
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
    const { result, unmount } = renderHook(
      () => ({ snapshot: useSnapshot(), autoSync: useAutoSync(true) }),
      { wrapper: createWrapper(queryClient) },
    );
    await waitFor(() => {
      expect(hookMocks.syncIfDue).toHaveBeenCalled();
      expect(result.current.autoSync.status.data).toBe(status);
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
  it("requests the debug report only once the panel asks for it", async () => {
    const report = {
      report: { problems: [] },
      text: "PRX diagnostic report\n",
    };
    hookMocks.getDebugReport.mockResolvedValue(report);
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });

    const disabled = renderHook(() => useDebugReport(false), {
      wrapper: createWrapper(queryClient),
    });
    expect(hookMocks.getDebugReport).not.toHaveBeenCalled();
    disabled.unmount();

    const { result } = renderHook(() => useDebugReport(true), {
      wrapper: createWrapper(queryClient),
    });
    await waitFor(() => {
      expect(result.current.data).toBe(report);
    });
    expect(hookMocks.getDebugReport).toHaveBeenCalledOnce();
  });

  it("reports the cached state of the shell queries without fetching them", async () => {
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });
    queryClient.setQueryData(["snapshot"], makeSnapshot());
    const { result } = renderHook(() => useQueryDiagnostics(), {
      wrapper: createWrapper(queryClient),
    });
    expect(result.current.map((query) => query.name)).toEqual([
      "snapshot",
      "github-config",
      "github-sync-status",
    ]);
    expect(result.current[0]?.state).toBe("success, idle");
    expect(result.current[1]?.state).toBe("not requested");
    expect(hookMocks.getSnapshot).not.toHaveBeenCalled();

    queryClient.setQueryData(["github-config"], undefined);
    await waitFor(() => {
      expect(hookMocks.getConfig).not.toHaveBeenCalled();
    });
  });
});
