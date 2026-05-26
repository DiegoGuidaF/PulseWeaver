import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  deleteDevicePairingMutation,
  getDevicesQueryKey,
  listDevicePairingsQueryKey,
} from "@/lib/api/@tanstack/react-query.gen";

export function useDeleteDevicePairing(deviceId: number) {
  const queryClient = useQueryClient();
  return useMutation({
    ...deleteDevicePairingMutation(),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: listDevicePairingsQueryKey({ path: { id: deviceId } }),
      });
      queryClient.invalidateQueries({ queryKey: getDevicesQueryKey() });
    },
  });
}
