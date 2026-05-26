import { useQuery } from "@tanstack/react-query";
import { listDevicePairingsOptions } from "@/lib/api/@tanstack/react-query.gen";

export function useListDevicePairings(deviceId: number, status: "pending" | "all" = "pending") {
  return useQuery({
    ...listDevicePairingsOptions({ path: { id: deviceId }, query: { status } }),
    staleTime: 0,
  });
}
