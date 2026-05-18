import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  demoteUserMutation,
  getCurrentUserQueryKey,
  listUsersQueryKey,
  listUsersWithAccessQueryKey,
} from "@/lib/api/@tanstack/react-query.gen";

export function useDemoteUser() {
  const queryClient = useQueryClient();
  return useMutation({
    ...demoteUserMutation(),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: listUsersQueryKey() });
      queryClient.invalidateQueries({ queryKey: listUsersWithAccessQueryKey() });
      queryClient.invalidateQueries({ queryKey: getCurrentUserQueryKey() });
    },
  });
}
