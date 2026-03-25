import { useQuery } from "@tanstack/react-query";
import { getDashboardTopDeniedIpsOptions } from "@/lib/api/@tanstack/react-query.gen";

export function useTopDeniedIPs(from?: string, to?: string, limit?: number) {
    return useQuery({
        ...getDashboardTopDeniedIpsOptions({ query: { from, to, limit } }),
        staleTime: 60_000,
    });
}
