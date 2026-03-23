import { useQuery, keepPreviousData } from "@tanstack/react-query";
import { getAddressHistoryOptions } from "@/lib/api/@tanstack/react-query.gen";
import type { GetAddressHistoryData } from "@/lib/api";

export function useAddressHistory(
  params: GetAddressHistoryData["query"],
  refetchInterval?: number | false,
) {
  return useQuery({
    ...getAddressHistoryOptions({ query: params }),
    staleTime: 10_000,
    placeholderData: keepPreviousData,
    refetchInterval: refetchInterval ?? false,
  });
}
