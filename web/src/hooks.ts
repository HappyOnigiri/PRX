import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect, useRef } from "react";
import {
  getConfig,
  getDebugReport,
  getSnapshot,
  getSyncStatus,
  syncIfDue,
} from "./api";
import type { QueryDiagnostic } from "./debug-text";

const snapshotKey = ["snapshot"] as const;
const configKey = ["github-config"] as const;
const syncStatusKey = ["github-sync-status"] as const;
const debugReportKey = ["debug-report"] as const;

// Returning the query object itself keeps React Query's property tracking
// intact. Spreading it would read every getter and make each consumer re-render
// on unrelated changes such as isFetching, which the automatic refresh below
// touches on every poll.
export function useSnapshot() {
  return useQuery({ queryKey: snapshotKey, queryFn: getSnapshot });
}

export function useConfig() {
  return useQuery({ queryKey: configKey, queryFn: getConfig });
}

// The report is a snapshot of the moment it was taken, so it is never refetched
// on its own. `enabled` keeps the request from firing until the debug tab is
// opened, because collecting the report reads the database and the config file.
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

// useQueryDiagnostics reports the cached state of the queries the shell keeps
// alive. It reads the cache rather than subscribing, so opening the debug tab
// never starts a fetch of its own and never hides a query that already failed.
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

export function useConfigMutation<TVariables, TData>(
  mutationFn: (input: TVariables) => Promise<TData>,
) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: configKey }),
  });
}
