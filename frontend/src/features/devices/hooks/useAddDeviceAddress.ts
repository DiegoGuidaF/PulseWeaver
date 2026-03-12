import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  addAddressMutation,
  getDeviceAddressesQueryKey,
} from "@/lib/api/@tanstack/react-query.gen";

export function useAddDeviceAddress(options?: { onSuccess?: () => void }) {
  const queryClient = useQueryClient();

  return useMutation({
    ...addAddressMutation(),
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({
        queryKey: getDeviceAddressesQueryKey({
          path: { device_id: variables.path.device_id },
        }),
      });
      options?.onSuccess?.();
    },
  });
}
