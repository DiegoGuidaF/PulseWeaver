import { useMutation, useQueryClient } from "@tanstack/react-query";
import { createDevice } from "@/lib/api";
import { queryKeys, toApiError, toErrorMessage } from "@/lib/api-client";
import { toast } from "sonner";
import type { CreateDeviceRequest, Device } from "@/lib/api";

export function useCreateDevice(options?: { onSuccess?: () => void }) {
  const queryClient = useQueryClient();

  return useMutation<Device, Error, CreateDeviceRequest>({
    mutationFn: async (values: CreateDeviceRequest) => {
      try {
        const response = await createDevice({
          body: values,
          throwOnError: false,
        });
        if (response.error) {
          throw toApiError(response.error);
        }
        return response.data;
      } catch (err) {
        throw toApiError(err);
      }
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.devices.all });
      toast.success("Device created", {
        description: "The new device has been added successfully.",
      });
      options?.onSuccess?.();
    },
    onError: (err) => {
      toast.error("Error creating device", {
        description: toErrorMessage(err),
      });
    },
  });
}
