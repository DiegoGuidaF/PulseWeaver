import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  updateDeviceMutation,
  getDevicesQueryKey,
} from "@/lib/api/@tanstack/react-query.gen";

export function useUpdateDevice() {
  const queryClient = useQueryClient();

  return useMutation({
    ...updateDeviceMutation(),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: getDevicesQueryKey() });
    },
  });
}
