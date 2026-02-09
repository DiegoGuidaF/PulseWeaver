import { useQuery } from "@tanstack/react-query";
import { api, toErrorMessage } from "@/lib/api/client";
import { queryKeys } from "@/lib/api/queryKeys";

export function useDeviceAddresses(deviceId: number, enabled: boolean) {
  return useQuery({
    queryKey: queryKeys.devices.addresses(deviceId),
    queryFn: async () => {
      const { data, error } = await api.GET("/devices/{device_id}/addresses", {
        params: { path: { device_id: deviceId } },
      });
      if (error) throw new Error(toErrorMessage(error));
      return data ?? [];
    },
    enabled,
  });
}
