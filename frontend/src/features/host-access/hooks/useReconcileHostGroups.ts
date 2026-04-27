import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  reconcileHostGroupsMutation,
  listHostGroupsQueryKey,
  listKnownHostsQueryKey,
  listUsersHostAccessQueryKey,
} from "@/lib/api/@tanstack/react-query.gen";

export function useReconcileHostGroups() {
  const queryClient = useQueryClient();
  return useMutation({
    ...reconcileHostGroupsMutation(),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: listHostGroupsQueryKey() });
      queryClient.invalidateQueries({ queryKey: listKnownHostsQueryKey() });
      queryClient.invalidateQueries({ queryKey: listUsersHostAccessQueryKey() });
      queryClient.invalidateQueries({ queryKey: [{ _id: "getUserHostDetails" }] });
    },
  });
}
