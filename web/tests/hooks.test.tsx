import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import {
  useAutoSync,
  useConfig,
  useConfigMutation,
  useDebugReport,
  useDisplayLanguage,
  useDomainMutation,
  useLanguageMutation,
  usePromptTemplates,
  usePromptTemplatesMutation,
  useQueryDiagnostics,
  useSnapshot,
  useSnapshotRefresh,
  useTaskLabelConfig,
  useTaskLabelConfigMutation,
  useUpdateStatus,
  useUpdateStatusInvalidation,
} from "../src/hooks";
import i18n from "../src/i18n";
import { readWebUISettings } from "../src/i18n/settings";
import { makeSnapshot } from "./factories";

const hookMocks = vi.hoisted(() => ({
  getSnapshot: vi.fn(),
  getDebugReport: vi.fn(),
  getConfig: vi.fn(),
  getSyncStatus: vi.fn(),
  syncIfDue: vi.fn(),
  getPromptTemplates: vi.fn(),
  getTaskLabelConfig: vi.fn(),
  getUpdateStatus: vi.fn(),
}));

vi.mock("../src/api", () => ({
  getSnapshot: hookMocks.getSnapshot,
  getDebugReport: hookMocks.getDebugReport,
  getConfig: hookMocks.getConfig,
  getSyncStatus: hookMocks.getSyncStatus,
  syncIfDue: hookMocks.syncIfDue,
  getPromptTemplates: hookMocks.getPromptTemplates,
  getTaskLabelConfig: hookMocks.getTaskLabelConfig,
  getUpdateStatus: hookMocks.getUpdateStatus,
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

  it("loads task label metadata and invalidates dependent queries after saving", async () => {
    const config = {
      keys: ["status.in_progress"],
      maxTextCodepoints: 32,
    };
    hookMocks.getTaskLabelConfig.mockResolvedValue(config);
    const mutation = vi.fn().mockResolvedValue({});
    const queryClient = new QueryClient({
      defaultOptions: {
        queries: { retry: false },
        mutations: { retry: false },
      },
    });
    const invalidateQueries = vi.spyOn(queryClient, "invalidateQueries");
    const { result } = renderHook(
      () => ({
        labels: useTaskLabelConfig(),
        save: useTaskLabelConfigMutation(mutation),
      }),
      { wrapper: createWrapper(queryClient) },
    );
    await waitFor(() => {
      expect(result.current.labels.data).toBe(config);
    });
    await act(async () => {
      await result.current.save.mutateAsync({ values: {} });
    });
    expect(mutation.mock.calls[0]?.[0]).toEqual({ values: {} });
    expect(invalidateQueries).toHaveBeenCalledWith({
      queryKey: ["task-label-config"],
    });
    expect(invalidateQueries).toHaveBeenCalledWith({ queryKey: ["snapshot"] });
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

  // 間引きはサーバーが持つので、クライアントはタブが前面へ戻るたびに読み直してよい。
  it("reloads the update status when the tab comes back", async () => {
    const status = {
      enabled: true,
      shouldNotify: true,
      latestVersion: "v0.5.0",
    };
    hookMocks.getUpdateStatus.mockResolvedValue(status);
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });
    const { result, unmount } = renderHook(() => useUpdateStatus(true), {
      wrapper: createWrapper(queryClient),
    });
    await waitFor(() => {
      expect(result.current.data).toBe(status);
    });
    window.dispatchEvent(new Event("focus"));
    await waitFor(() => {
      expect(hookMocks.getUpdateStatus.mock.calls.length).toBeGreaterThan(1);
    });
    unmount();
  });

  it("does not read the update status when it is disabled", () => {
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });
    renderHook(() => useUpdateStatus(false), {
      wrapper: createWrapper(queryClient),
    });
    expect(hookMocks.getUpdateStatus).not.toHaveBeenCalled();
  });

  it("drops the cached update status after a skip or an install", async () => {
    const queryClient = new QueryClient();
    const invalidateQueries = vi.spyOn(queryClient, "invalidateQueries");
    const { result } = renderHook(() => useUpdateStatusInvalidation(), {
      wrapper: createWrapper(queryClient),
    });
    await result.current();
    expect(invalidateQueries).toHaveBeenCalledWith({
      queryKey: ["update-status"],
    });
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

  it("drops the cached snapshot on demand", async () => {
    const queryClient = new QueryClient();
    const invalidateQueries = vi.spyOn(queryClient, "invalidateQueries");
    const { result } = renderHook(() => useSnapshotRefresh(), {
      wrapper: createWrapper(queryClient),
    });

    await result.current();
    expect(invalidateQueries).toHaveBeenCalledWith({ queryKey: ["snapshot"] });
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
  it("reads prompt templates and refreshes them after a write", async () => {
    const templates = { design: "Design {{task_id}}", implementation: "" };
    hookMocks.getPromptTemplates.mockResolvedValue(templates);
    const queryClient = new QueryClient({
      defaultOptions: {
        queries: { retry: false },
        mutations: { retry: false },
      },
    });
    const invalidateQueries = vi.spyOn(queryClient, "invalidateQueries");

    const { result } = renderHook(() => usePromptTemplates(), {
      wrapper: createWrapper(queryClient),
    });
    await waitFor(() => {
      expect(result.current.data).toBe(templates);
    });

    const mutation = vi.fn().mockResolvedValue("saved");
    const mutationHook = renderHook(
      () => usePromptTemplatesMutation(mutation),
      {
        wrapper: createWrapper(queryClient),
      },
    );
    await expect(
      mutationHook.result.current.mutateAsync("input"),
    ).resolves.toBe("saved");
    expect(invalidateQueries).toHaveBeenCalledWith({
      queryKey: ["prompt-templates"],
    });
  });

  // 言語を変えると組み込みテンプレートも変わるので、設定とテンプレートの
  // 両方のキャッシュを捨てる。
  it("refreshes the configuration and the templates after a language write", async () => {
    const queryClient = new QueryClient({
      defaultOptions: {
        queries: { retry: false },
        mutations: { retry: false },
      },
    });
    const invalidateQueries = vi.spyOn(queryClient, "invalidateQueries");

    const mutation = vi.fn().mockResolvedValue("saved");
    const { result } = renderHook(() => useLanguageMutation(mutation), {
      wrapper: createWrapper(queryClient),
    });
    await expect(result.current.mutateAsync("ja")).resolves.toBe("saved");
    expect(invalidateQueries).toHaveBeenCalledWith({
      queryKey: ["github-config"],
    });
    expect(invalidateQueries).toHaveBeenCalledWith({
      queryKey: ["prompt-templates"],
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
      "revision-stream",
    ]);
    // 購読は shell が開くので、context のない debug 表示では未開始になる。
    expect(result.current[3]?.state).toBe("not started");
    expect(result.current[0]?.state).toBe("success, idle");
    expect(result.current[1]?.state).toBe("not requested");
    expect(hookMocks.getSnapshot).not.toHaveBeenCalled();

    queryClient.setQueryData(["github-config"], undefined);
    await waitFor(() => {
      expect(hookMocks.getConfig).not.toHaveBeenCalled();
    });
  });
});

// 表示言語の正はサーバーの設定である。auto のときもサーバーが解決した実効言語に
// 従うので、画面の言語とプロンプトの言語がずれない。
describe("display language", () => {
  beforeEach(async () => {
    localStorage.clear();
    await i18n.changeLanguage("en");
  });

  it("follows the effective language the server resolved", async () => {
    hookMocks.getConfig.mockResolvedValue({
      language: "auto",
      effectiveLanguage: "ja",
    });
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });

    renderHook(
      () => {
        useDisplayLanguage();
      },
      { wrapper: createWrapper(queryClient) },
    );
    await waitFor(() => {
      expect(i18n.resolvedLanguage).toBe("ja");
    });
    // Local Storage は初回描画のためのキャッシュとして更新される。
    expect(readWebUISettings().language).toBe("ja");
  });

  it("keeps the current language when the server reports an unknown one", async () => {
    hookMocks.getConfig.mockResolvedValue({
      language: "auto",
      effectiveLanguage: "fr",
    });
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });

    const { result } = renderHook(
      () => {
        useDisplayLanguage();
        return useConfig();
      },
      { wrapper: createWrapper(queryClient) },
    );
    await waitFor(() => {
      expect(result.current.data).toBeDefined();
    });
    expect(i18n.resolvedLanguage).toBe("en");
  });
});
