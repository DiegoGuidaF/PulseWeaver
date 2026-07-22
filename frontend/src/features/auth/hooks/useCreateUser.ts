import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  createUserMutation,
  listUsersQueryKey,
  listUsersWithAccessQueryKey,
} from "@/lib/api/@tanstack/react-query.gen";
import { invalidateOwnerIdentity } from "@/features/devices/fleetCache";

export function useCreateUser() {
  const queryClient = useQueryClient();

  return useMutation({
    ...createUserMutation(),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: listUsersQueryKey() });
      queryClient.invalidateQueries({ queryKey: listUsersWithAccessQueryKey() });
      // A new user is an owner the devices page must offer as a create target.
      invalidateOwnerIdentity(queryClient);
    },
  });
}
