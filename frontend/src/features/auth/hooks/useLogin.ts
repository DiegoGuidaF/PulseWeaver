import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "react-router";
import {
  getCurrentUserQueryKey,
  loginMutation,
} from "@/lib/api/@tanstack/react-query.gen";
import { safeReturnTo } from "@/lib/routes";

export function useLogin() {
  const queryClient = useQueryClient();
  const navigate = useNavigate();

  return useMutation({
    ...loginMutation(),
    onSuccess: async () => {
      // Invalidate and wait for auth state to update before navigating
      // This ensures ProtectedRoute sees the updated auth state immediately
      await queryClient.invalidateQueries({ queryKey: getCurrentUserQueryKey() });
      const params = new URLSearchParams(window.location.search);
      navigate(safeReturnTo(params.get("returnTo")));
    },
  });
}
