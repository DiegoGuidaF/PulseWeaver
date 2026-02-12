import { useQuery } from "@tanstack/react-query";
import { api, toApiError } from "@/lib/api/client";
import { queryKeys } from "@/lib/api/queryKeys";

export function useCurrentUser() {
  const query = useQuery({
    queryKey: queryKeys.auth.currentUser,
    queryFn: async () => {
      const { data, error } = await api.GET("/auth/me");
      // 401 means not authenticated, which is a valid state (not an error)
      if (error) {
        // Check if error has status property (openapi-fetch error structure)
        const status = (error as any)?.status;
        if (status === 401) {
          // Not authenticated - return null (valid state)
          return null;
        }
        // Other errors should be thrown with status preserved
        throw toApiError(error);
      }
      return data ?? null;
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
