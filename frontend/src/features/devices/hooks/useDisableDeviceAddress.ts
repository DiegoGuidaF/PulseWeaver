import { useMutation, useQueryClient } from "@tanstack/react-query";
import { api, toErrorMessage } from "@/lib/api/client";
import { queryKeys } from "@/lib/api/queryKeys";
import { toast } from "sonner";
import type { Address } from "@/lib/api/types";

export function useDisableDeviceAddress(deviceId: number) {
  const queryClient = useQueryClient();

  return useMutation<Address, Error, number>({
    mutationFn: async (addressId: number) => {
      const { data, error } = await api.DELETE(
        "/devices/{device_id}/addresses/{address_id}",
        {
          params: {
            path: { device_id: deviceId, address_id: addressId },
          },
        },
      );
      if (error) throw new Error(toErrorMessage(error));
      return data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: queryKeys.devices.addresses(deviceId),
      });
      toast.success("Address disabled");
    },
    onError: (err) => {
      toast.error("Error disabling address", { description: err.message });
    },
  });
}
