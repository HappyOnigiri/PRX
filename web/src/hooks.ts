import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { getSnapshot } from "./api";

const snapshotKey = ["snapshot"] as const;

export function useSnapshot() {
  return useQuery({ queryKey: snapshotKey, queryFn: getSnapshot });
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
