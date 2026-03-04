import { useQuery } from "@tanstack/react-query";
import { getCurrentUser } from "@/lib/api";
import { getCurrentUserQueryKey } from "@/lib/api/@tanstack/react-query.gen";
import { toApiError } from "@/lib/api-client";
import type { User } from "@/lib/api";

export function useCurrentUser() {
  const query = useQuery<User | null>({
    queryKey: getCurrentUserQueryKey(),
    queryFn: async () => {
      try {
        const response = await getCurrentUser({ throwOnError: false });
        if (response.error) {
          if (response.response?.status === 401) return null;
          throw toApiError(response.error);
        }
        return response.data ?? null;
      } catch (err) {
        const status =
          err && typeof err === 'object' && 'status' in err
            ? Number((err as { status: unknown }).status)
            : undefined;
        if (status === 401) return null;
        throw toApiError(err);
      }
    },
    retry: false,
    // Session validation: refetch when user returns to tab
    refetchOnWindowFocus: true,
    // Consider session valid for 5 minutes
    staleTime: 5 * 60 * 1000,
    // Always check on mount (explicit, though it's default)
    refetchOnMount: true,
  });

  return {
    data: query.data ?? null,
    isLoading: query.isLoading,
    isAuthenticated: !!query.data,
    error: query.error,
  };
}
