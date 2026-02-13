import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  disableAddressMutation,
  getDeviceAddressesQueryKey,
} from "@/lib/api/@tanstack/react-query.gen";
import { toErrorMessage } from "@/lib/api-client";
import { toast } from "sonner";

export function useDisableDeviceAddress() {
  const queryClient = useQueryClient();

  return useMutation({
    ...disableAddressMutation(),
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({
        queryKey: getDeviceAddressesQueryKey({
          path: { device_id: variables.path.device_id },
        }),
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
