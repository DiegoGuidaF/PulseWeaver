import { useQuery } from "@tanstack/react-query";
import { getRequestAuditLogDenyReasonsOptions } from "@/lib/api/@tanstack/react-query.gen";

export function useRequestAuditLogDenyReasons() {
    return useQuery(getRequestAuditLogDenyReasonsOptions());
}
