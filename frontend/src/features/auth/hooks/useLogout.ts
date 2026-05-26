import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "react-router-dom";
import { logoutMutation, getCurrentUserQueryKey } from "@/lib/api/@tanstack/react-query.gen";
import { ROUTES } from "@/lib/routes";

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
      navigate(ROUTES.login, { replace: true });
    },
  });
}
