import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect, useRef } from "react";
import { getConfig, getSnapshot, getSyncStatus, syncIfDue } from "./api";

const snapshotKey = ["snapshot"] as const;
const configKey = ["github-config"] as const;
const syncStatusKey = ["github-sync-status"] as const;

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
