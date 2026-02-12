import { useQuery } from "@tanstack/react-query";
import { getCurrentUser } from "@/lib/api";
import { queryKeys, toApiError } from "@/lib/api-client";
import type { User } from "@/lib/api";

export function useCurrentUser() {
  const query = useQuery<User | null>({
    queryKey: queryKeys.auth.currentUser,
    queryFn: async () => {
      try {
        const response = await getCurrentUser({ throwOnError: false });
        // 401 means not authenticated, which is a valid state (not an error)
        if (response.error) {
          const status = (response.error as any)?.status;
          if (status === 401) {
            // Not authenticated - return null (valid state)
            return null;
          }
          // Other errors should be thrown with status preserved
          throw toApiError(response.error);
        }
        return response.data ?? null;
      } catch (err) {
        const status = (err as any)?.status;
        if (status === 401) {
          return null;
        }
        throw toApiError(err);
      }
    },
    // Never retry - 401 is expected when not logged in, other errors shouldn't retry either
    retry: false,
  });

  return {
    data: query.data ?? null,
    isLoading: query.isLoading,
    isAuthenticated: !!query.data,
    error: query.error,
  };
}
