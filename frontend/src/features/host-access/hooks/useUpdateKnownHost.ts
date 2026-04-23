import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  updateKnownHostMutation,
  listKnownHostsQueryKey,
} from "@/lib/api/@tanstack/react-query.gen";

export function useUpdateKnownHost() {
  const queryClient = useQueryClient();
  return useMutation({
    ...updateKnownHostMutation(),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: listKnownHostsQueryKey() });
    },
  });
}
