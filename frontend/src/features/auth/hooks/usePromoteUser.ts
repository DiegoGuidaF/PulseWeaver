import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  promoteUserMutation,
  getCurrentUserQueryKey,
  getUserAccessDetailQueryKey,
  listUsersQueryKey,
  listUsersWithAccessQueryKey,
} from "@/lib/api/@tanstack/react-query.gen";
import type { Options, PromoteUserData } from "@/lib/api";

export function usePromoteUser() {
  const queryClient = useQueryClient();
  return useMutation({
    ...promoteUserMutation(),
    onSuccess: (_data, variables: Options<PromoteUserData>) => {
      queryClient.invalidateQueries({ queryKey: listUsersQueryKey() });
      queryClient.invalidateQueries({ queryKey: listUsersWithAccessQueryKey() });
      queryClient.invalidateQueries({ queryKey: getCurrentUserQueryKey() });
      queryClient.invalidateQueries({
        queryKey: getUserAccessDetailQueryKey({ path: { user_id: variables.path!.user_id } }),
      });
    },
  });
}
