import { useQuery, keepPreviousData } from "@tanstack/react-query";
import { getAddressHistoryHistogramOptions } from "@/lib/api/@tanstack/react-query.gen";
import type { GetAddressHistoryHistogramData } from "@/lib/api";

/** Event-count buckets for the chart. Filters + window only — no cursor in the key, so page turns on the events list never refetch this. */
export function useAddressHistoryHistogram(
    params: GetAddressHistoryHistogramData["query"],
    refetchInterval: number | false = false,
) {
    return useQuery({
        ...getAddressHistoryHistogramOptions({ query: params }),
        staleTime: 10_000,
        placeholderData: keepPreviousData,
        refetchInterval,
    });
}
