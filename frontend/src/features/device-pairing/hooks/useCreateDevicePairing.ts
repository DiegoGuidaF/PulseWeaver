import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  createDevicePairingMutation,
  getDevicesQueryKey,
  listDevicePairingsQueryKey,
} from "@/lib/api/@tanstack/react-query.gen";

export function useCreateDevicePairing(deviceId: number) {
  const queryClient = useQueryClient();
  return useMutation({
    ...createDevicePairingMutation(),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: listDevicePairingsQueryKey({ path: { id: deviceId } }),
      });
      queryClient.invalidateQueries({ queryKey: getDevicesQueryKey() });
    },
  });
}
