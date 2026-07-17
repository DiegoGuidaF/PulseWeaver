import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  setUserAccessMutation,
  getUserAccessDetailQueryKey,
  listUsersWithAccessQueryKey,
} from "@/lib/api/@tanstack/react-query.gen";
import type { Options, SetUserAccessData } from "@/lib/api";

export function useSetUserAccess() {
  const queryClient = useQueryClient();

  return useMutation({
    ...setUserAccessMutation(),
    onSuccess: (_data, variables: Options<SetUserAccessData>) => {
      queryClient.invalidateQueries({
        queryKey: getUserAccessDetailQueryKey({ path: { user_id: variables.path!.user_id } }),
      });
      queryClient.invalidateQueries({ queryKey: listUsersWithAccessQueryKey() });
      // Partial-key invalidation: a user's group grant changes that group's users[]
      // panel; we don't know which groups were touched, so bust every cached detail.
      queryClient.invalidateQueries({ queryKey: [{ _id: "getHostGroup" }] });
    },
  });
}
