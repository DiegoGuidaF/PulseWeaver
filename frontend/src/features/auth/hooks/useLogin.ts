import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "react-router-dom";
import { login } from "@/lib/api";
import { queryKeys, toApiError, toErrorMessage } from "@/lib/api-client";
import { toast } from "sonner";
import type { AuthRequest, User } from "@/lib/api";

export function useLogin() {
  const queryClient = useQueryClient();
  const navigate = useNavigate();

  return useMutation<User, Error, AuthRequest>({
    mutationFn: async (values: AuthRequest) => {
      try {
        const response = await login({
          body: values,
          throwOnError: false,
        });
        if (response.error) {
          throw toApiError(response.error);
        }
        return response.data;
      } catch (err) {
        throw toApiError(err);
      }
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.auth.currentUser });
      // Get returnTo from URL or default to /devices
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
