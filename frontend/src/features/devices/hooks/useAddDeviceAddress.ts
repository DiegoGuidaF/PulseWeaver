import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  addAddressMutation,
  getDeviceAddressesQueryKey,
} from "@/lib/api/@tanstack/react-query.gen";
import { toErrorMessage } from "@/lib/api-client";
import { toast } from "sonner";

export function useAddDeviceAddress(
  _deviceId: number,
  options?: { onSuccess?: () => void },
) {
  const queryClient = useQueryClient();

  return useMutation({
    ...addAddressMutation(),
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({
        queryKey: getDeviceAddressesQueryKey({
          path: { device_id: variables.path.device_id },
        }),
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
