import { useMutation, useQueryClient } from "@tanstack/react-query";
import { regenerateDeviceApiKeyMutation } from "@/lib/api/@tanstack/react-query.gen";
import { getDeviceQueryKey } from "@/lib/api/@tanstack/react-query.gen";
import { toast } from "sonner";
import { toErrorMessage } from "@/lib/api-client";

export function useRegenerateApiKey() {
  const queryClient = useQueryClient();

  return useMutation({
    ...regenerateDeviceApiKeyMutation(),
    onSuccess: (data, variables) => {
      queryClient.setQueryData(
        getDeviceQueryKey({ path: { device_id: variables.path.device_id } }),
        data.device,
      );
      toast.success("API key regenerated");
    },
    onError: (error) => {
      toast.error(toErrorMessage(error));
    },
  });
}
