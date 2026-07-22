import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  setUserAccessMutation,
  getUserAccessDetailQueryKey,
  listUsersWithAccessQueryKey,
} from "@/lib/api/@tanstack/react-query.gen";
import type { Options, SetUserAccessData } from "@/lib/api";
import { invalidateOwnerIdentity } from "@/features/devices/fleetCache";

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
      // bypass_host_check and group_ids are rendered as owner badges on the device
      // surfaces, which cache their own copy behind FLEET_STALE_TIME.
      invalidateOwnerIdentity(queryClient);
    },
  });
}
