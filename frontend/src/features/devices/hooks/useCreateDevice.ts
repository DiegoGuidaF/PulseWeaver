import { useMutation, useQueryClient } from "@tanstack/react-query";
import { api, toErrorMessage } from "@/lib/api/client";
import { queryKeys } from "@/lib/api/queryKeys";
import { toast } from "sonner";

export function useCreateDevice(options?: { onSuccess?: () => void }) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (values: { name: string }) => {
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
