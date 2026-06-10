import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  demoteUserMutation,
  getCurrentUserQueryKey,
  getUserAccessDetailQueryKey,
  listUsersQueryKey,
  listUsersWithAccessQueryKey,
} from "@/lib/api/@tanstack/react-query.gen";
import type { Options, DemoteUserData } from "@/lib/api";

export function useDemoteUser() {
  const queryClient = useQueryClient();
  return useMutation({
    ...demoteUserMutation(),
    onSuccess: (_data, variables: Options<DemoteUserData>) => {
      queryClient.invalidateQueries({ queryKey: listUsersQueryKey() });
      queryClient.invalidateQueries({ queryKey: listUsersWithAccessQueryKey() });
      queryClient.invalidateQueries({ queryKey: getCurrentUserQueryKey() });
      queryClient.invalidateQueries({
        queryKey: getUserAccessDetailQueryKey({ path: { user_id: variables.path!.user_id } }),
      });
    },
  });
}
