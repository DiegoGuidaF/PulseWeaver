import { useQuery, keepPreviousData } from "@tanstack/react-query";
import { getAddressHistoryOptions } from "@/lib/api/@tanstack/react-query.gen";
import type { GetAddressHistoryData } from "@/lib/api";

/** Cursor-paginated address events. The cursor is part of the query key, so page turns refetch this and only this. */
export function useAddressHistory(
    params: GetAddressHistoryData["query"],
    refetchInterval: number | false = false,
) {
    return useQuery({
        ...getAddressHistoryOptions({ query: params }),
        staleTime: 10_000,
        placeholderData: keepPreviousData,
        refetchInterval,
    });
}
