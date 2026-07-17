import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  reconcileHostGroupsMutation,
  listHostGroupsQueryKey,
  listHostSuggestionsQueryKey,
  listHostsQueryKey,
  listUsersWithAccessQueryKey,
  listNetworkPoliciesQueryKey,
} from "@/lib/api/@tanstack/react-query.gen";

export function useReconcileHostGroups() {
  const queryClient = useQueryClient();
  return useMutation({
    ...reconcileHostGroupsMutation(),
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: listHostGroupsQueryKey() }),
        // Partial-key invalidation: every cached group's own detail may have changed
        // (renamed, recolored, membership) — the mutation doesn't return which ids.
        queryClient.invalidateQueries({ queryKey: [{ _id: "getHostGroup" }] }),
        queryClient.invalidateQueries({ queryKey: listHostsQueryKey() }),
        queryClient.invalidateQueries({ queryKey: listHostSuggestionsQueryKey() }),
        queryClient.invalidateQueries({ queryKey: listUsersWithAccessQueryKey() }),
        // host_count and member lists on any network policy may have moved.
        queryClient.invalidateQueries({ queryKey: listNetworkPoliciesQueryKey() }),
        queryClient.invalidateQueries({ queryKey: [{ _id: "getNetworkPolicy" }] }),
        // Partial-key invalidation: invalidates getUserAccessDetail for all user IDs.
        // A host/group reconcile can shift effective access for any subject-group
        // member; the response carries no affected-user list to narrow this further.
        queryClient.invalidateQueries({ queryKey: [{ _id: "getUserAccessDetail" }] }),
      ]);
    },
  });
}
