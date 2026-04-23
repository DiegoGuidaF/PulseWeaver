import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  setUserHostGrantsMutation,
  listUsersHostAccessQueryKey,
  getUserHostDetailsQueryKey,
} from "@/lib/api/@tanstack/react-query.gen";

export function useSetUserHostGrants(userId: number) {
  const queryClient = useQueryClient();
  return useMutation({
    ...setUserHostGrantsMutation(),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: listUsersHostAccessQueryKey() });
      queryClient.invalidateQueries({
        queryKey: getUserHostDetailsQueryKey({ path: { user_id: userId } }),
      });
    },
  });
}
