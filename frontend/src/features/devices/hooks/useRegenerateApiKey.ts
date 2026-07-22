import { useMutation, useQueryClient } from "@tanstack/react-query";
import { regenerateDeviceApiKeyMutation } from "@/lib/api/@tanstack/react-query.gen";
import { invalidateOwnerFleet } from "../fleetCache";

export function useRegenerateApiKey(ownerId: number) {
  const queryClient = useQueryClient();

  return useMutation({
    ...regenerateDeviceApiKeyMutation(),
    onSuccess: () => {
      invalidateOwnerFleet(queryClient, ownerId);
    },
  });
}
