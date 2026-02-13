import { useMutation, useQueryClient } from "@tanstack/react-query";
import { logoutMutation } from "@/lib/api/@tanstack/react-query.gen";
import { toErrorMessage } from "@/lib/api-client";
import { toast } from "sonner";

export function useLogout() {
  const queryClient = useQueryClient();

  return useMutation({
    ...logoutMutation(),
    onSuccess: () => {
      queryClient.clear();
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
