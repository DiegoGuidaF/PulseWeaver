import { useMutation, useQueryClient } from "@tanstack/react-query";
import { logout } from "@/lib/api";
import { toast } from "sonner";
import { toApiError, toErrorMessage } from "@/lib/api-client";

export function useLogout() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async () => {
      try {
        const response = await logout({ throwOnError: false });
        if (response.error) {
          throw toApiError(response.error);
        }
      } catch (err) {
        throw toApiError(err);
      }
    },
    onSuccess: () => {
      // Clear all queries to reset app state
      queryClient.clear();
      // Redirect will be handled by the global 401 handler if needed,
      // but we'll also do it here to ensure immediate redirect
      window.location.href = "/login";
      toast.success("Logged out", {
        description: "You have been logged out successfully.",
      });
    },
    onError: (err) => {
      toast.error("Logout failed", {
        description: toErrorMessage(err),
      });
    },
  });
}
