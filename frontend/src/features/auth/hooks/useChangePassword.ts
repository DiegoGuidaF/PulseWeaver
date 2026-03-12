import { useMutation, useQueryClient } from "@tanstack/react-query";
import { getCurrentUserQueryKey, changePasswordMutation } from "@/lib/api/@tanstack/react-query.gen";

export function useChangePassword() {
  const queryClient = useQueryClient();

  return useMutation({
    ...changePasswordMutation(),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: getCurrentUserQueryKey() });
    },
  });
}
