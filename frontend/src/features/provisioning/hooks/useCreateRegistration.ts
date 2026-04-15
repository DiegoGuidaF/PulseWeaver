import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  createRegistrationMutation,
  listRegistrationsQueryKey,
} from "@/lib/api/@tanstack/react-query.gen";

export function useCreateRegistration() {
  const queryClient = useQueryClient();
  return useMutation({
    ...createRegistrationMutation(),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: listRegistrationsQueryKey() });
    },
  });
}
