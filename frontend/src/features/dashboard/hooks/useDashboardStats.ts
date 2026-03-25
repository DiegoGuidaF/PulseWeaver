import { useQuery } from "@tanstack/react-query";
import { getDashboardStatsOptions } from "@/lib/api/@tanstack/react-query.gen";

export function useDashboardStats(from?: string, to?: string) {
    return useQuery({
        ...getDashboardStatsOptions({ query: { from, to } }),
        staleTime: 60_000,
    });
}
