import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  createHostGroupMutation,
  listHostGroupsQueryKey,
} from "@/lib/api/@tanstack/react-query.gen";

export function useCreateHostGroup() {
  const queryClient = useQueryClient();
  return useMutation({
    ...createHostGroupMutation(),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: listHostGroupsQueryKey() });
    },
  });
}
