import { useQuery } from "@tanstack/react-query";
import { listAnomaliesOptions } from "@/lib/api/@tanstack/react-query.gen";
import type { ListAnomaliesData } from "@/lib/api";

type AnomaliesQuery = NonNullable<ListAnomaliesData["query"]>;

/** Scan cadence is 5 minutes; a ~90s refetch keeps the list fresh without over-polling. */
const REFETCH_INTERVAL_MS = 90_000;

export function useAnomalies(query: AnomaliesQuery) {
    return useQuery({
        ...listAnomaliesOptions({ query }),
        refetchInterval: REFETCH_INTERVAL_MS,
    });
}
