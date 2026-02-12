import { useMutation, useQueryClient } from "@tanstack/react-query";
import { api, toErrorMessage } from "@/lib/api/client";
import { queryKeys } from "@/lib/api/queryKeys";
import { toast } from "sonner";
import type { CreateDeviceRequest, Device } from "@/lib/api/types";

export function useCreateDevice(options?: { onSuccess?: () => void }) {
  const queryClient = useQueryClient();

  return useMutation<Device, Error, CreateDeviceRequest>({
    mutationFn: async (values: CreateDeviceRequest) => {
      const { data, error } = await api.POST("/devices", {
        body: values,
      });
      if (error) throw new Error(toErrorMessage(error));
      return data;
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
        description: err.message,
      });
    },
  });
}
