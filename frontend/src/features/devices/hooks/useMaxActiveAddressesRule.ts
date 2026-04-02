import { useQuery } from "@tanstack/react-query";
import { getMaxActiveAddressesRuleQueryKey } from "@/lib/api/@tanstack/react-query.gen";
import { getMaxActiveAddressesRule } from "@/lib/api";
import type { MaxActiveAddressesRule } from "@/lib/api";

const pathOptions = (deviceId: number) => ({ path: { device_id: deviceId } });

export function useMaxActiveAddressesRule(
  deviceId: number,
  enabled = true,
): {
  data: MaxActiveAddressesRule | null;
  isLoading: boolean;
  isError: boolean;
  error: unknown;
} {
  const queryKey = getMaxActiveAddressesRuleQueryKey(pathOptions(deviceId));
  const result = useQuery({
    queryKey,
    queryFn: async ({ signal }) => {
      const response = await getMaxActiveAddressesRule({
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
