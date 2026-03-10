import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  deleteUserMutation,
  getCurrentUserQueryKey,
  listUsersQueryKey,
} from "@/lib/api/@tanstack/react-query.gen";
import { toErrorMessage } from "@/lib/api-client";
import { toast } from "sonner";

export function useDeleteUser() {
  const queryClient = useQueryClient();

  return useMutation({
    ...deleteUserMutation(),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: listUsersQueryKey() });
      queryClient.invalidateQueries({ queryKey: getCurrentUserQueryKey() });
      toast.success("User deleted");
    },
    onError: (err) => {
      toast.error("Failed to delete user", {
        description: toErrorMessage(err),
      });
    },
  });
}
