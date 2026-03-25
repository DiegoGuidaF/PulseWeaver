import { useQuery } from "@tanstack/react-query";
import { getDashboardServicesOptions } from "@/lib/api/@tanstack/react-query.gen";

export function useDashboardServices(from?: string, to?: string) {
    return useQuery({
        ...getDashboardServicesOptions({ query: { from, to } }),
        staleTime: 60_000,
    });
}
