import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  createDeviceMutation,
  getDevicesQueryKey,
} from "@/lib/api/@tanstack/react-query.gen";
import type { CreateDeviceResponse } from "@/lib/api";

export function useCreateDevice(options?: {
  onSuccess?: (data: CreateDeviceResponse) => void;
}) {
  const queryClient = useQueryClient();

  return useMutation({
    ...createDeviceMutation(),
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: getDevicesQueryKey() });
      options?.onSuccess?.(data);
    },
  });
}
