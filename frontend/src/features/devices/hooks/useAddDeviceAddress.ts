import { useMutation, useQueryClient } from "@tanstack/react-query";
import { api, toErrorMessage } from "@/lib/api/client";
import { queryKeys } from "@/lib/api/queryKeys";
import { toast } from "sonner";

export function useAddDeviceAddress(
  deviceId: number,
  options?: { onSuccess?: () => void },
) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (ip: string) => {
      const { data, error } = await api.POST("/devices/{device_id}/addresses", {
        params: { path: { device_id: deviceId } },
        body: { ip },
      });
      if (error) throw new Error(toErrorMessage(error));
      return data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: queryKeys.devices.addresses(deviceId),
      });
      toast.success("Address added");
      options?.onSuccess?.();
    },
    onError: (err) => {
      toast.error("Error adding address", { description: err.message });
    },
  });
}
