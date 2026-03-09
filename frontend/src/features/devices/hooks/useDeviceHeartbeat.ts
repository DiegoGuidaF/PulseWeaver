import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  deviceHeartbeatMutation,
  getDeviceAddressesQueryKey,
} from "@/lib/api/@tanstack/react-query.gen";
import { toErrorMessage } from "@/lib/api-client";
import { toast } from "sonner";

export function useDeviceHeartbeat() {
  const queryClient = useQueryClient();
  return useMutation({
    ...deviceHeartbeatMutation(),
    onSuccess: (address, variables) => {
      queryClient.invalidateQueries({
        queryKey: getDeviceAddressesQueryKey({ path: { device_id: variables.path.device_id } }),
      });
      toast.success(`IP ${address.ip} registered`);
    },
    onError: (err) => toast.error("Heartbeat failed", { description: toErrorMessage(err) }),
  });
}
