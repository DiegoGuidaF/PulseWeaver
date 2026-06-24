import { useMutation, useQueryClient } from "@tanstack/react-query";
import { logoutMutation, getCurrentUserQueryKey } from "@/lib/api/@tanstack/react-query.gen";

export function useLogout() {
  const queryClient = useQueryClient();

  return useMutation({
    ...logoutMutation(),
    onSuccess: () => {
      const meKey = getCurrentUserQueryKey();
      // Mark the session logged out: ProtectedRoute sees isAuthenticated:false
      // and redirects to /login. This must run before dropping the rest of the
      // cache — queryClient.clear() detaches the useCurrentUser observer, so a
      // null written afterwards never reaches it and no redirect fires.
      queryClient.setQueryData(meKey, null);
      // Drop every other cached query so a subsequent login can't briefly read
      // the previous user's data.
      queryClient.removeQueries({
        predicate: (query) =>
          JSON.stringify(query.queryKey) !== JSON.stringify(meKey),
      });
    },
  });
}
