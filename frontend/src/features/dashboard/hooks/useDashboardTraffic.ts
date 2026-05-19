import { useQuery } from "@tanstack/react-query";
import { getDashboardTrafficOptions } from "@/lib/api/@tanstack/react-query.gen";

export function useDashboardTraffic(from?: string, to?: string) {
    return useQuery({
        ...getDashboardTrafficOptions({ query: { from, to } }),
        staleTime: 60_000,
    });
}
