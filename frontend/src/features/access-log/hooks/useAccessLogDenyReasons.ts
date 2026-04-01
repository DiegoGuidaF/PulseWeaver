import { useQuery } from "@tanstack/react-query";
import { getAccessLogDenyReasonsOptions } from "@/lib/api/@tanstack/react-query.gen";

export function useAccessLogDenyReasons() {
    return useQuery(getAccessLogDenyReasonsOptions());
}
