import { useQuery } from "@tanstack/react-query";
import { getAccessLogByCountryOptions } from "@/lib/api/@tanstack/react-query.gen";

export function useCountryStats(from?: string, to?: string) {
    return useQuery({
        ...getAccessLogByCountryOptions({ query: { from, to } }),
        staleTime: 60_000,
    });
}
