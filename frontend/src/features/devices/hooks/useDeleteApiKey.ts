import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  deleteDeviceApiKeyMutation,
  getDevicesQueryKey,
} from "@/lib/api/@tanstack/react-query.gen";

export function useDeleteApiKey() {
  const queryClient = useQueryClient();

  return useMutation({
    ...deleteDeviceApiKeyMutation(),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: getDevicesQueryKey() });
    },
  });
}
