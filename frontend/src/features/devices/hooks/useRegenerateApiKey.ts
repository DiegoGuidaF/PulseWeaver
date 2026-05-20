import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  regenerateDeviceApiKeyMutation,
  getDevicesQueryKey,
} from "@/lib/api/@tanstack/react-query.gen";

export function useRegenerateApiKey() {
  const queryClient = useQueryClient();

  return useMutation({
    ...regenerateDeviceApiKeyMutation(),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: getDevicesQueryKey() });
    },
  });
}
