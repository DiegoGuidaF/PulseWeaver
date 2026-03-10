import { useMutation, useQueryClient } from "@tanstack/react-query";
import { getCurrentUserQueryKey, changePasswordMutation } from "@/lib/api/@tanstack/react-query.gen";
import { toErrorMessage } from "@/lib/api-client";
import { toast } from "sonner";

export function useChangePassword() {
  const queryClient = useQueryClient();

  return useMutation({
    ...changePasswordMutation(),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: getCurrentUserQueryKey() });
      toast.success("Password changed");
    },
    onError: (err) => {
      toast.error("Failed to change password", {
        description: toErrorMessage(err),
      });
    },
  });
}
