import { useQuery } from "@tanstack/react-query";
import { getDeviceAddresses } from "@/lib/api";
import { queryKeys, toApiError } from "@/lib/api-client";
import type { Address } from "@/lib/api";

export function useDeviceAddresses(deviceId: number, enabled: boolean) {
  return useQuery<Address[]>({
    queryKey: queryKeys.devices.addresses(deviceId),
    queryFn: async () => {
      try {
        const response = await getDeviceAddresses({
          path: { device_id: deviceId },
          throwOnError: false,
        });
        if (response.error) {
          throw toApiError(response.error);
        }
        return response.data ?? [];
      } catch (err) {
        throw toApiError(err);
      }
    },
    enabled,
  });
}
