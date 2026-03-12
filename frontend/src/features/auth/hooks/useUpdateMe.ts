import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  getCurrentUserQueryKey,
  updateMeMutation,
} from "@/lib/api/@tanstack/react-query.gen";

export function useUpdateMe() {
  const queryClient = useQueryClient();

  return useMutation({
    ...updateMeMutation(),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: getCurrentUserQueryKey() });
    },
  });
}
