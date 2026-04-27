import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  reconcileKnownHostsMutation,
  listKnownHostsQueryKey,
  listHostGroupsQueryKey,
  listUsersHostAccessQueryKey,
} from "@/lib/api/@tanstack/react-query.gen";

export function useReconcileKnownHosts() {
  const queryClient = useQueryClient();
  return useMutation({
    ...reconcileKnownHostsMutation(),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: listKnownHostsQueryKey() });
      queryClient.invalidateQueries({ queryKey: listHostGroupsQueryKey() });
      queryClient.invalidateQueries({ queryKey: listUsersHostAccessQueryKey() });
      queryClient.invalidateQueries({ queryKey: [{ _id: "getUserHostDetails" }] });
    },
  });
}
