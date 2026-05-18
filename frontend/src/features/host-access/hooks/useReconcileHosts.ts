import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  reconcileHostsMutation,
  listHostsQueryKey,
  listHostGroupsQueryKey,
  listHostSuggestionsQueryKey,
  listUsersWithAccessQueryKey,
} from "@/lib/api/@tanstack/react-query.gen";

export function useReconcileHosts() {
  const queryClient = useQueryClient();
  return useMutation({
    ...reconcileHostsMutation(),
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: listHostsQueryKey() }),
        queryClient.invalidateQueries({ queryKey: listHostGroupsQueryKey() }),
        queryClient.invalidateQueries({ queryKey: listHostSuggestionsQueryKey() }),
        queryClient.invalidateQueries({ queryKey: listUsersWithAccessQueryKey() }),
        // Partial-key invalidation: invalidates getUserAccessDetail for all user IDs
        queryClient.invalidateQueries({ queryKey: [{ _id: "getUserAccessDetail" }] }),
      ]);
    },
  });
}
