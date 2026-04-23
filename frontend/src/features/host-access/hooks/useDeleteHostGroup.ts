import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  deleteHostGroupMutation,
  listHostGroupsQueryKey,
  listUsersHostAccessQueryKey,
} from "@/lib/api/@tanstack/react-query.gen";

export function useDeleteHostGroup() {
  const queryClient = useQueryClient();
  return useMutation({
    ...deleteHostGroupMutation(),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: listHostGroupsQueryKey() });
      queryClient.invalidateQueries({ queryKey: listUsersHostAccessQueryKey() });
      queryClient.invalidateQueries({ queryKey: [{ _id: "getUserHostDetails" }] });
    },
  });
}
