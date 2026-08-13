import { useQuery } from "@tanstack/react-query";
import { getAccessLogEntryOptions } from "@/lib/api/@tanstack/react-query.gen";
import { toApiError } from "@/lib/api-client";

/**
 * The full record behind one access-log row, for the detail drawer. Never
 * polled: an entry is immutable once written, so a fetched detail cannot go
 * stale — it can only disappear when retention prunes it, which surfaces as a
 * 404 rather than as changed content.
 */
export function useAccessLogEntry(id: number | null) {
    return useQuery({
        ...getAccessLogEntryOptions({ path: { id: id ?? 0 } }),
        enabled: id != null,
        staleTime: Infinity,
        // A pruned or unknown id is a settled answer, not a transient failure;
        // retrying it only delays the "no longer available" state.
        retry: (failureCount, error) => toApiError(error).status !== 404 && failureCount < 3,
    });
}

/** True when the entry is gone — unknown id, or aged out of retention. */
export function isEntryUnavailable(error: unknown): boolean {
    return toApiError(error).status === 404;
}
