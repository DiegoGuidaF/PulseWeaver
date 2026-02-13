import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "react-router-dom";
import {
  getCurrentUserQueryKey,
  loginMutation,
} from "@/lib/api/@tanstack/react-query.gen";
import { toErrorMessage } from "@/lib/api-client";
import { toast } from "sonner";

export function useLogin() {
  const queryClient = useQueryClient();
  const navigate = useNavigate();

  return useMutation({
    ...loginMutation(),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: getCurrentUserQueryKey() });
      const params = new URLSearchParams(window.location.search);
      const returnTo = params.get("returnTo") || "/devices";
      navigate(returnTo);
      toast.success("Login successful", {
        description: "You have been logged in successfully.",
      });
    },
    onError: (err) => {
      toast.error("Login failed", {
        description: toErrorMessage(err),
      });
    },
  });
}
