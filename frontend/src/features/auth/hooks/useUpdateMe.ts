import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  getCurrentUserQueryKey,
  updateMeMutation,
} from "@/lib/api/@tanstack/react-query.gen";
import { toApiError, toErrorMessage } from "@/lib/api-client";
import { toast } from "sonner";

export function useUpdateMe() {
  const queryClient = useQueryClient();

  return useMutation({
    ...updateMeMutation(),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: getCurrentUserQueryKey() });
      toast.success("Profile updated");
    },
    onError: (err) => {
      const apiErr = toApiError(err);
      const description =
        apiErr.status === 409
          ? "Username or email is already in use."
          : toErrorMessage(err);
      toast.error("Failed to update profile", { description });
    },
  });
}
