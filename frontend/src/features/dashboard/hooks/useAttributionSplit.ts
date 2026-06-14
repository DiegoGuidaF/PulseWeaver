import { useQuery } from "@tanstack/react-query";
import { getDashboardAttributionSplitOptions } from "@/lib/api/@tanstack/react-query.gen";
import type { GetDashboardAttributionSplitData } from "@/lib/api";

export type AttributionKind = GetDashboardAttributionSplitData["query"]["kind"];

export function useAttributionSplit(kind: AttributionKind, from?: string, to?: string) {
    return useQuery({
        ...getDashboardAttributionSplitOptions({ query: { kind, from, to } }),
        staleTime: 60_000,
    });
}
