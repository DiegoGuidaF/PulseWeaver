import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  regenerateDeviceApiKeyMutation,
  getDeviceQueryKey,
} from "@/lib/api/@tanstack/react-query.gen";

export function useRegenerateApiKey() {
  const queryClient = useQueryClient();

  return useMutation({
    ...regenerateDeviceApiKeyMutation(),
    onSuccess: (data, variables) => {
      queryClient.setQueryData(
        getDeviceQueryKey({ path: { device_id: variables.path.device_id } }),
        data.device,
      );
    },
  });
}
