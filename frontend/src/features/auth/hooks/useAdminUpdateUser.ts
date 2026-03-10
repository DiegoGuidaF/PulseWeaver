import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  adminUpdateUserMutation,
  getCurrentUserQueryKey,
  listUsersQueryKey,
} from "@/lib/api/@tanstack/react-query.gen";
import { toErrorMessage } from "@/lib/api-client";
import { toast } from "sonner";

export function useAdminUpdateUser() {
  const queryClient = useQueryClient();

  return useMutation({
    ...adminUpdateUserMutation(),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: listUsersQueryKey() });
      queryClient.invalidateQueries({ queryKey: getCurrentUserQueryKey() });
      toast.success("User updated");
    },
    onError: (err) => {
      toast.error("Failed to update user", {
        description: toErrorMessage(err),
      });
    },
  });
}
