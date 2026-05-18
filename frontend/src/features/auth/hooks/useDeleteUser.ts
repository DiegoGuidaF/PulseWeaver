import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  deleteUserMutation,
  getCurrentUserQueryKey,
  listUsersQueryKey,
  listUsersWithAccessQueryKey,
} from "@/lib/api/@tanstack/react-query.gen";

export function useDeleteUser() {
  const queryClient = useQueryClient();

  return useMutation({
    ...deleteUserMutation(),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: listUsersQueryKey() });
      queryClient.invalidateQueries({ queryKey: listUsersWithAccessQueryKey() });
      queryClient.invalidateQueries({ queryKey: getCurrentUserQueryKey() });
    },
  });
}
