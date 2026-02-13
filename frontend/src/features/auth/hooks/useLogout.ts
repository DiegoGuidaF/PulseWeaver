import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "react-router-dom";
import { logoutMutation } from "@/lib/api/@tanstack/react-query.gen";
import { toErrorMessage } from "@/lib/api-client";
import { toast } from "sonner";

export function useLogout() {
  const queryClient = useQueryClient();
  const navigate = useNavigate();

  return useMutation({
    ...logoutMutation(),
    onSuccess: () => {
      // Clear all queries (logout = fresh start)
      queryClient.removeQueries();
      // Show toast before navigation
      toast.success("Logged out", {
        description: "You have been logged out successfully.",
      });
      // Use React Router navigate for consistency with login
      navigate("/login");
    },
    onError: (err) => {
      toast.error("Logout failed", {
        description: toErrorMessage(err),
      });
    },
  });
}
