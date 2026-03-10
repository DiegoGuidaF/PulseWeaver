import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "react-router-dom";
import { logoutMutation, getCurrentUserQueryKey } from "@/lib/api/@tanstack/react-query.gen";
import { toErrorMessage } from "@/lib/api-client";
import { toast } from "sonner";

export function useLogout() {
  const queryClient = useQueryClient();
  const navigate = useNavigate();

  return useMutation({
    ...logoutMutation(),
    onSuccess: () => {
      // Clear all caches, then immediately set user to null so LoginPage
      // sees isAuthenticated:false (not a loading state) before the navigate
      // lands — preventing it from redirecting straight back to /devices.
      queryClient.clear();
      queryClient.setQueryData(getCurrentUserQueryKey(), null);
      navigate("/login", { replace: true });
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
