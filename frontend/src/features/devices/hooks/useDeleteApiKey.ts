import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  deleteDeviceApiKeyMutation,
  getDeviceQueryKey,
} from "@/lib/api/@tanstack/react-query.gen";

export function useDeleteApiKey() {
  const queryClient = useQueryClient();

  return useMutation({
    ...deleteDeviceApiKeyMutation(),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({
        queryKey: getDeviceQueryKey({
          path: { device_id: variables.path.device_id },
        }),
      });
    },
  });
}
