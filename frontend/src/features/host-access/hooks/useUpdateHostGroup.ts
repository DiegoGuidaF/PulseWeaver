import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  updateHostGroupMutation,
  listHostGroupsQueryKey,
} from "@/lib/api/@tanstack/react-query.gen";

export function useUpdateHostGroup() {
  const queryClient = useQueryClient();
  return useMutation({
    ...updateHostGroupMutation(),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: listHostGroupsQueryKey() });
      queryClient.invalidateQueries({ queryKey: [{ _id: "getUserHostDetails" }] });
    },
  });
}
