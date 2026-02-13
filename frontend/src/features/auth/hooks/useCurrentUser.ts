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
          const status = (response.error as { status?: number })?.status;
          if (status === 401) return null;
          throw toApiError(response.error);
        }
        return response.data ?? null;
      } catch (err) {
        const status = (err as { status?: number })?.status;
        if (status === 401) return null;
        throw toApiError(err);
      }
    },
    retry: false,
  });

  return {
    data: query.data ?? null,
    isLoading: query.isLoading,
    isAuthenticated: !!query.data,
    error: query.error,
  };
}
