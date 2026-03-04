import { useQuery } from "@tanstack/react-query";
import { getDeviceAddressLeaseRuleQueryKey } from "@/lib/api/@tanstack/react-query.gen";
import { getDeviceAddressLeaseRule } from "@/lib/api";
import type { DeviceAddressLeaseRule } from "@/lib/api";

const pathOptions = (deviceId: number) => ({ path: { device_id: deviceId } });

export function useDeviceAddressLeaseRule(
  deviceId: number,
  enabled = true,
): {
  data: DeviceAddressLeaseRule | null;
  isLoading: boolean;
  isError: boolean;
  error: unknown;
} {
  const queryKey = getDeviceAddressLeaseRuleQueryKey(pathOptions(deviceId));
  const result = useQuery({
    queryKey,
    queryFn: async ({ signal }) => {
      const response = await getDeviceAddressLeaseRule({
        ...pathOptions(deviceId),
        signal,
        throwOnError: false,
      });
      if (response.data !== undefined) return response.data;
      if (
        "response" in response &&
        response.response &&
        response.response.status === 404
      ) {
        return null;
      }
      throw response.error;
    },
    enabled,
  });

  return {
    data: result.data ?? null,
    isLoading: result.isLoading,
    isError: result.isError,
    error: result.error,
  };
}
