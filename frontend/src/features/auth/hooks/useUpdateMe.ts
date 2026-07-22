import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  getCurrentUserQueryKey,
  updateMeMutation,
} from "@/lib/api/@tanstack/react-query.gen";
import { invalidateOwnerIdentity } from "@/features/devices/fleetCache";

export function useUpdateMe() {
  const queryClient = useQueryClient();

  return useMutation({
    ...updateMeMutation(),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: getCurrentUserQueryKey() });
      // Display name is rendered on the owner card and in every owner picker.
      invalidateOwnerIdentity(queryClient);
    },
  });
}
