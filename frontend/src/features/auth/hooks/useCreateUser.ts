import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  createUserMutation,
  listUsersQueryKey,
  listUsersWithAccessQueryKey,
} from "@/lib/api/@tanstack/react-query.gen";

export function useCreateUser() {
  const queryClient = useQueryClient();

  return useMutation({
    ...createUserMutation(),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: listUsersQueryKey() });
      queryClient.invalidateQueries({ queryKey: listUsersWithAccessQueryKey() });
    },
  });
}
