import { useMutation, useQueryClient } from "@tanstack/react-query";
import { api, toApiError } from "@/lib/api/client";
import { toast } from "sonner";

export function useLogout() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async () => {
      const { error } = await api.POST("/auth/logout");
      if (error) throw toApiError(error);
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
        description: err.message,
      });
    },
  });
}
