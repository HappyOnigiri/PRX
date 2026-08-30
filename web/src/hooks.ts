import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { getConfig, getSnapshot } from "./api";

const snapshotKey = ["snapshot"] as const;
const configKey = ["github-config"] as const;

export function useSnapshot() {
  return useQuery({ queryKey: snapshotKey, queryFn: getSnapshot });
}

export function useConfig() {
  return useQuery({ queryKey: configKey, queryFn: getConfig });
}

export function useDomainMutation<TVariables, TData>(
  mutationFn: (input: TVariables) => Promise<TData>,
) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: snapshotKey }),
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
