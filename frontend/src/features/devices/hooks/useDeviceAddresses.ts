import { useQuery } from "@tanstack/react-query";
import { getDeviceAddressesOptions } from "@/lib/api/@tanstack/react-query.gen";

export function useDeviceAddresses(
  deviceId: number,
  enabled = true,
  refetchInterval: number | false = false,
) {
  return useQuery({
    ...getDeviceAddressesOptions({ path: { device_id: deviceId } }),
    enabled,
    refetchInterval,
  });
}
