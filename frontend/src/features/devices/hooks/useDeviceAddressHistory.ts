import { useQuery } from "@tanstack/react-query";
import { getDeviceAddressHistoryOptions } from "@/lib/api/@tanstack/react-query.gen";

export function useDeviceAddressHistory(
  deviceId: number,
  from?: string,
  to?: string,
  granularity?: string,
) {
  return useQuery({
    ...getDeviceAddressHistoryOptions({
      path: { device_id: deviceId },
      query: { from, to, granularity },
    }),
  });
}
