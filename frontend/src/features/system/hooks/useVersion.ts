import { useQuery } from "@tanstack/react-query";
import { getVersionOptions } from "@/lib/api/@tanstack/react-query.gen";

// The running binary cannot change under a live session, so this never needs
// refetching once loaded.
export function useVersion() {
    return useQuery({
        ...getVersionOptions(),
        staleTime: Infinity,
        gcTime: Infinity,
        retry: false,
    });
}
