import { useQuery } from "@tanstack/react-query";
import { api, toErrorMessage } from "@/lib/api/client";
import { queryKeys } from "@/lib/api/queryKeys";
import type { Device } from "@/lib/api/types";

export function useDevices() {
  return useQuery<Device[]>({
    queryKey: queryKeys.devices.all,
    queryFn: async () => {
      const { data, error } = await api.GET("/devices");
      if (error) throw new Error(toErrorMessage(error));
      return data ?? [];
    },
  });
}
