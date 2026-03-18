import { useQuery, keepPreviousData } from "@tanstack/react-query";
import { getRequestAuditLogOptions } from "@/lib/api/@tanstack/react-query.gen";
import type { GetRequestAuditLogData } from "@/lib/api";

export function useRequestAuditLog(params: GetRequestAuditLogData["query"]) {
    return useQuery({
        ...getRequestAuditLogOptions({ query: params }),
        staleTime: 10_000,
        placeholderData: keepPreviousData,
    });
}
