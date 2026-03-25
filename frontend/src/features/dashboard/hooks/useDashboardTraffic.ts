import { useQuery } from "@tanstack/react-query";
import { getDashboardTrafficOptions } from "@/lib/api/@tanstack/react-query.gen";
import type { GetDashboardTrafficData } from "@/lib/api";

export function useDashboardTraffic(
    from?: string,
    to?: string,
    granularity?: NonNullable<GetDashboardTrafficData["query"]>["granularity"],
) {
    return useQuery({
        ...getDashboardTrafficOptions({ query: { from, to, granularity } }),
        staleTime: 60_000,
    });
}
