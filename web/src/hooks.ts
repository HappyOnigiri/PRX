import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useCallback, useEffect, useRef } from "react";
import {
  getConfig,
  getDebugReport,
  getPromptTemplates,
  getSnapshot,
  getSyncStatus,
  syncIfDue,
} from "./api";
import type { QueryDiagnostic } from "./debug-text";

const snapshotKey = ["snapshot"] as const;
const configKey = ["github-config"] as const;
const promptTemplatesKey = ["prompt-templates"] as const;
const syncStatusKey = ["github-sync-status"] as const;
const debugReportKey = ["debug-report"] as const;

// query オブジェクトをそのまま返して React Query のプロパティ追跡を保つ。分解す
// ると全 getter を読むため、ポーリングのたびに変わる isFetching のような無関係な
// 変化でも利用側が再描画される。
export function useSnapshot() {
  return useQuery({ queryKey: snapshotKey, queryFn: getSnapshot });
}

export function useConfig() {
  return useQuery({ queryKey: configKey, queryFn: getConfig });
}

// テンプレートは設定パネルでしか読まないので、shell が保持し続ける query には
// 含めず、パネルが初めてマウントされたときに読み込む。
export function usePromptTemplates() {
  return useQuery({
    queryKey: promptTemplatesKey,
    queryFn: getPromptTemplates,
  });
}

export function usePromptTemplatesMutation<TVariables, TData>(
  mutationFn: (input: TVariables) => Promise<TData>,
) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn,
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: promptTemplatesKey }),
  });
}

// レポートは取得時点のスナップショットなので、自動で再取得はしない。収集時に
// データベースと設定ファイルを読むため、`enabled` で debug タブが開かれるまで
// リクエストを発行しない。
export function useDebugReport(enabled: boolean) {
  return useQuery({
    queryKey: debugReportKey,
    queryFn: getDebugReport,
    enabled,
    staleTime: Infinity,
    gcTime: 0,
  });
}

export function useAutoSync(enabled = true) {
  const queryClient = useQueryClient();
  const checking = useRef(false);
  const status = useQuery({
    queryKey: syncStatusKey,
    queryFn: getSyncStatus,
    enabled,
  });
  const check = useMutation({
    mutationFn: syncIfDue,
    onMutate: () => {
      checking.current = true;
    },
    onSuccess: async (response) => {
      if (response.status)
        queryClient.setQueryData(syncStatusKey, response.status);
      if (response.ran)
        await queryClient.invalidateQueries({ queryKey: snapshotKey });
    },
    onSettled: () => {
      checking.current = false;
    },
  });
  const { mutate } = check;

  useEffect(() => {
    if (!enabled) return;
    const run = () => {
      if (document.visibilityState === "visible" && !checking.current) mutate();
    };
    run();
    const interval = window.setInterval(run, 60_000);
    document.addEventListener("visibilitychange", run);
    window.addEventListener("focus", run);
    return () => {
      window.clearInterval(interval);
      document.removeEventListener("visibilitychange", run);
      window.removeEventListener("focus", run);
    };
  }, [enabled, mutate]);

  return { status, checking: check.isPending, error: check.error };
}

// useQueryDiagnostics は shell が保持する query のキャッシュ状態を返す。購読では
// なくキャッシュを読むので、debug タブを開いても新たな取得は始まらず、すでに失敗
// している query も隠れない。
export function useQueryDiagnostics(): QueryDiagnostic[] {
  const queryClient = useQueryClient();
  return [snapshotKey, configKey, syncStatusKey].map((key) => {
    const state = queryClient.getQueryState(key);
    if (!state) return { name: key[0], state: "not requested" };
    if (state.error)
      return { name: key[0], state: `error: ${state.error.message}` };
    return { name: key[0], state: `${state.status}, ${state.fetchStatus}` };
  });
}

export function useDomainMutation<TVariables, TData>(
  mutationFn: (input: TVariables) => Promise<TData>,
) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn,
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: snapshotKey }),
        queryClient.invalidateQueries({ queryKey: syncStatusKey }),
      ]);
    },
  });
}

// 部分的に書き込んだところで失敗した mutation は onSuccess を通らないので、
// 書き込めた分をグラフへ出すために呼び出し側からスナップショットを捨てる。
export function useSnapshotRefresh() {
  const queryClient = useQueryClient();
  return useCallback(
    () => queryClient.invalidateQueries({ queryKey: snapshotKey }),
    [queryClient],
  );
}

export function useConfigMutation<TVariables, TData>(
  mutationFn: (input: TVariables) => Promise<TData>,
) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: configKey }),
  });
}
