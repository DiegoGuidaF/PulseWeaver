import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  reconcileHostsMutation,
  listHostsQueryKey,
  listHostGroupsQueryKey,
  listHostSuggestionsQueryKey,
  listUsersWithAccessQueryKey,
  listNetworkPoliciesQueryKey,
} from "@/lib/api/@tanstack/react-query.gen";
import { invalidateOwnerIdentity } from "@/features/devices/fleetCache";

export function useReconcileHosts() {
  const queryClient = useQueryClient();
  return useMutation({
    ...reconcileHostsMutation(),
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: listHostsQueryKey() }),
        queryClient.invalidateQueries({ queryKey: listHostGroupsQueryKey() }),
        // Partial-key invalidation: a host moving groups changes hosts[] on every
        // cached group detail; the mutation doesn't tell us which ids, so all of them.
        queryClient.invalidateQueries({ queryKey: [{ _id: "getHostGroup" }] }),
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
      // Group membership shifts change the owner badges the fleet renders from its
      // own cached copy of host_groups[].
      invalidateOwnerIdentity(queryClient);
    },
  });
}
