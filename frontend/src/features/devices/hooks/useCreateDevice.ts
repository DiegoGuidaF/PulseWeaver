import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  createDeviceMutation,
  getDevicesQueryKey,
} from "@/lib/api/@tanstack/react-query.gen";
import type { CreateDeviceResponse } from "@/lib/api";
import { toApiError, toErrorMessage } from "@/lib/api-client";
import { toast } from "sonner";

export function useCreateDevice(options?: {
  onSuccess?: (data: CreateDeviceResponse) => void;
}) {
  const queryClient = useQueryClient();

  return useMutation({
    ...createDeviceMutation(),
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: getDevicesQueryKey() });
      toast.success("Device created", {
        description: "The new device has been added successfully.",
      });
      options?.onSuccess?.(data);
    },
    onError: (err) => {
      const apiErr = toApiError(err);
      const description =
        apiErr.status === 409
          ? "A device with this name already exists."
          : toErrorMessage(err);
      toast.error("Error creating device", {
        description,
      });
    },
  });
}
