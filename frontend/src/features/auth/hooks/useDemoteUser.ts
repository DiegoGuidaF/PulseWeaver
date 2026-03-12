import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  demoteUserMutation,
  getCurrentUserQueryKey,
  listUsersQueryKey,
} from "@/lib/api/@tanstack/react-query.gen";

export function useDemoteUser() {
  const queryClient = useQueryClient();
  return useMutation({
    ...demoteUserMutation(),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: listUsersQueryKey() });
      queryClient.invalidateQueries({ queryKey: getCurrentUserQueryKey() });
    },
  });
}
