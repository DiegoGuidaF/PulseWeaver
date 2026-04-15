import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  deleteRegistrationMutation,
  listRegistrationsQueryKey,
} from "@/lib/api/@tanstack/react-query.gen";

export function useDeleteRegistration() {
  const queryClient = useQueryClient();
  return useMutation({
    ...deleteRegistrationMutation(),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: listRegistrationsQueryKey() });
    },
  });
}
