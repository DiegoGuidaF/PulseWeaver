import { useQuery } from "@tanstack/react-query";
import { PairingListFilter } from "@/lib/api";
import { listDevicePairingsOptions } from "@/lib/api/@tanstack/react-query.gen";

export function useListDevicePairings(
  deviceId: number,
  status: PairingListFilter = PairingListFilter.PENDING,
) {
  return useQuery({
    ...listDevicePairingsOptions({ path: { id: deviceId }, query: { status } }),
    staleTime: 0,
  });
}
