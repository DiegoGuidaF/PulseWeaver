import { useQuery, keepPreviousData } from "@tanstack/react-query";
import { getAddressHistoryTuningOptions } from "@/lib/api/@tanstack/react-query.gen";
import type { GetAddressHistoryTuningData } from "@/lib/api";

/**
 * Devices whose lease TTL is shorter than their real check-in cadence.
 * Window-scoped by contract — the endpoint takes no column filters, so slicing
 * the events table never re-ranks the fleet.
 */
export function useAddressHistoryTuning(
    params: GetAddressHistoryTuningData["query"],
    refetchInterval: number | false = false,
) {
    return useQuery({
        ...getAddressHistoryTuningOptions({ query: params }),
        staleTime: 10_000,
        placeholderData: keepPreviousData,
        refetchInterval,
    });
}
