import { useQuery, keepPreviousData } from "@tanstack/react-query";
import { getAccessLogOptions } from "@/lib/api/@tanstack/react-query.gen";
import type { GetAccessLogData } from "@/lib/api";

export function useAccessLog(
    params: GetAccessLogData["query"],
    refetchInterval: number | false = false,
) {
    return useQuery({
        ...getAccessLogOptions({ query: params }),
        staleTime: 10_000,
        placeholderData: keepPreviousData,
        refetchInterval,
    });
}
