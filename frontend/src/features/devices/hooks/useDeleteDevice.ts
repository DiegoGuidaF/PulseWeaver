import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  deleteDeviceMutation,
  getDevicesQueryKey,
} from "@/lib/api/@tanstack/react-query.gen";
import { toErrorMessage } from "@/lib/api-client";
import { toast } from "sonner";

export function useDeleteDevice(options?: { onSuccess?: () => void }) {
  const queryClient = useQueryClient();

  return useMutation({
    ...deleteDeviceMutation(),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: getDevicesQueryKey() });
      toast.success("Device deleted", {
        description: "The device has been removed and will no longer appear in the list.",
      });
      options?.onSuccess?.();
    },
    onError: (err) => {
      toast.error("Error deleting device", {
        description: toErrorMessage(err),
      });
    },
  });
}
