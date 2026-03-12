import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  adminUpdateUserMutation,
  getCurrentUserQueryKey,
  listUsersQueryKey,
} from "@/lib/api/@tanstack/react-query.gen";

export function useAdminUpdateUser() {
  const queryClient = useQueryClient();

  return useMutation({
    ...adminUpdateUserMutation(),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: listUsersQueryKey() });
      queryClient.invalidateQueries({ queryKey: getCurrentUserQueryKey() });
    },
  });
}
