import { useMutation, useQueryClient } from "@tanstack/react-query";
import { addAddress } from "@/lib/api";
import { queryKeys, toApiError, toErrorMessage } from "@/lib/api-client";
import { toast } from "sonner";
import type { Address } from "@/lib/api";

export function useAddDeviceAddress(
  deviceId: number,
  options?: { onSuccess?: () => void },
) {
  const queryClient = useQueryClient();

  return useMutation<Address, Error, string>({
    mutationFn: async (ip: string) => {
      try {
        const response = await addAddress({
          path: { device_id: deviceId },
          body: { ip },
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
      toast.success("Address added");
      options?.onSuccess?.();
    },
    onError: (err) => {
      toast.error("Error adding address", {
        description: toErrorMessage(err),
      });
    },
  });
}
