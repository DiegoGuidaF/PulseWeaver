import { useMutation, useQueryClient } from "@tanstack/react-query";
import { disableAddress } from "@/lib/api";
import { queryKeys, toApiError, toErrorMessage } from "@/lib/api-client";
import { toast } from "sonner";
import type { Address } from "@/lib/api";

export function useDisableDeviceAddress(deviceId: number) {
  const queryClient = useQueryClient();

  return useMutation<Address, Error, number>({
    mutationFn: async (addressId: number) => {
      try {
        const response = await disableAddress({
          path: { device_id: deviceId, address_id: addressId },
          throwOnError: false,
        });
        if (response.error) {
          throw toApiError(response.error);
        }
        return response.data;
      } catch (err) {
        throw toApiError(err);
      }
    },
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: queryKeys.devices.addresses(deviceId),
      });
      toast.success("Address disabled");
    },
    onError: (err) => {
      toast.error("Error disabling address", {
        description: toErrorMessage(err),
      });
    },
  });
}
