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
      // Partial key match: invalidates all getDevicesByUser queries regardless of user_id.
      // TanStack Query v5 partialDeepEqual matches any stored key whose first element
      // contains _id: 'getDevicesByUser', so no cast or predicate needed.
      queryClient.invalidateQueries({ queryKey: [{ _id: "getDevicesByUser" }] });
      options?.onSuccess?.(data);
    },
  });
}
