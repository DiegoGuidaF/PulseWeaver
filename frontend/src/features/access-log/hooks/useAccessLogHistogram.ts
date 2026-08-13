import { useQuery, keepPreviousData } from "@tanstack/react-query";
import { getAccessLogHistogramOptions } from "@/lib/api/@tanstack/react-query.gen";
import type { GetAccessLogHistogramData } from "@/lib/api";

/** Allow/deny buckets for the chart above the table. Filters + window only — no cursor in the key, so page turns never re-scan the histogram. */
export function useAccessLogHistogram(
    params: GetAccessLogHistogramData["query"],
    refetchInterval: number | false = false,
) {
    return useQuery({
        ...getAccessLogHistogramOptions({ query: params }),
        staleTime: 10_000,
        placeholderData: keepPreviousData,
        refetchInterval,
    });
}
