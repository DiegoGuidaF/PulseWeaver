import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  deleteKnownHostMutation,
  listKnownHostsQueryKey,
  listUsersHostAccessQueryKey,
} from "@/lib/api/@tanstack/react-query.gen";

export function useDeleteKnownHost() {
  const queryClient = useQueryClient();
  return useMutation({
    ...deleteKnownHostMutation(),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: listKnownHostsQueryKey() });
      queryClient.invalidateQueries({ queryKey: listUsersHostAccessQueryKey() });
    },
  });
}
