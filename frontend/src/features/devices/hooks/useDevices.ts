import { useQuery } from "@tanstack/react-query";
import { getDevices } from "@/lib/api";
import { queryKeys, toApiError, toErrorMessage } from "@/lib/api-client";
import type { Device } from "@/lib/api";

export function useDevices() {
  return useQuery<Device[]>({
    queryKey: queryKeys.devices.all,
    queryFn: async () => {
      try {
        const response = await getDevices({ throwOnError: false });
        if (response.error) {
          throw toApiError(response.error);
        }
        return response.data ?? [];
      } catch (err) {
        throw toApiError(err);
      }
    },
  });
}
