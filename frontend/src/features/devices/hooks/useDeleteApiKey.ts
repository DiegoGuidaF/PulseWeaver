import { useMutation, useQueryClient } from "@tanstack/react-query";
import { deleteDeviceApiKeyMutation } from "@/lib/api/@tanstack/react-query.gen";
import { invalidateOwnerFleet } from "../fleetCache";

export function useDeleteApiKey(ownerId: number) {
  const queryClient = useQueryClient();

  return useMutation({
    ...deleteDeviceApiKeyMutation(),
    onSuccess: () => {
      invalidateOwnerFleet(queryClient, ownerId);
    },
  });
}
