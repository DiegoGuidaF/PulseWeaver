import { useMutation, useQueryClient } from "@tanstack/react-query";
import { enableDeviceMutation } from "@/lib/api/@tanstack/react-query.gen";
import { invalidateOwnerFleet } from "../fleetCache";

export function useEnableDevice(ownerId: number) {
  const queryClient = useQueryClient();

  return useMutation({
    ...enableDeviceMutation(),
    onSuccess: () => {
      invalidateOwnerFleet(queryClient, ownerId);
    },
  });
}
