import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  promoteUserMutation,
  getCurrentUserQueryKey,
  listUsersQueryKey,
  listUsersWithAccessQueryKey,
} from "@/lib/api/@tanstack/react-query.gen";

export function usePromoteUser() {
  const queryClient = useQueryClient();
  return useMutation({
    ...promoteUserMutation(),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: listUsersQueryKey() });
      queryClient.invalidateQueries({ queryKey: listUsersWithAccessQueryKey() });
      queryClient.invalidateQueries({ queryKey: getCurrentUserQueryKey() });
    },
  });
}
