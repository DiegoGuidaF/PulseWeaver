import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  createDeviceMutation,
  getDevicesQueryKey,
} from "@/lib/api/@tanstack/react-query.gen";
import { toErrorMessage } from "@/lib/api-client";
import { toast } from "sonner";

export function useCreateDevice(options?: { onSuccess?: () => void }) {
  const queryClient = useQueryClient();

  return useMutation({
    ...createDeviceMutation(),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: getDevicesQueryKey() });
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
