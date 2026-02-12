import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "react-router-dom";
import { api, toApiError } from "@/lib/api/client";
import { queryKeys } from "@/lib/api/queryKeys";
import { toast } from "sonner";
import type { AuthRequest, User } from "@/lib/api/types";

export function useLogin() {
  const queryClient = useQueryClient();
  const navigate = useNavigate();

  return useMutation<User, Error, AuthRequest>({
    mutationFn: async (values: AuthRequest) => {
      const { data, error } = await api.POST("/auth/login", {
        body: values,
      });
      if (error) throw toApiError(error);
      return data;
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
        description: err.message,
      });
    },
  });
}
