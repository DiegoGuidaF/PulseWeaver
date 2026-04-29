import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  reconcileHostGroupsMutation,
  listHostGroupsQueryKey,
  listHostSuggestionsQueryKey,
  listKnownHostsQueryKey,
  listUsersHostAccessQueryKey,
} from "@/lib/api/@tanstack/react-query.gen";

export function useReconcileHostGroups() {
  const queryClient = useQueryClient();
  return useMutation({
    ...reconcileHostGroupsMutation(),
    onSuccess: async () => {
      // Await all invalidations so the mutation's pending state (and any awaiting
      // mutateAsync caller) covers the post-save refetch. This way the page-level
      // isFetching loaders stay aligned with the Save button's spinner.
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: listHostGroupsQueryKey() }),
        queryClient.invalidateQueries({ queryKey: listKnownHostsQueryKey() }),
        queryClient.invalidateQueries({ queryKey: listHostSuggestionsQueryKey() }),
        queryClient.invalidateQueries({ queryKey: listUsersHostAccessQueryKey() }),
        queryClient.invalidateQueries({ queryKey: [{ _id: "getUserHostDetails" }] }),
      ]);
    },
  });
}
