import { useQuery, useQueryClient } from "@tanstack/react-query";
import {
  getDeviceQueryKey,
  getDevicesQueryKey,
} from "@/lib/api/@tanstack/react-query.gen";
import {
  getDevice,
  type Device,
  type GetDevicesResponse,
} from "@/lib/api";

function findDeviceById(
  devices: GetDevicesResponse | undefined,
  deviceId: number,
): Device | undefined {
  return devices?.find((device) => device.id === deviceId);
}

export function useDevice(deviceId: number, refetchInterval: number | false = false) {
  const queryClient = useQueryClient();

  return useQuery({
    queryKey: getDeviceQueryKey({
      path: { device_id: deviceId },
    }),
    queryFn: async ({ signal }) => {
      const response = await getDevice({
        path: { device_id: deviceId },
        signal,
        throwOnError: false,
      });

      if (response.data !== undefined) {
        return response.data;
      }

      if (
        "response" in response &&
        response.response &&
        response.response.status === 404
      ) {
        return undefined;
      }

      throw response.error;
    },
    initialData: () => {
      const devices = queryClient.getQueryData<GetDevicesResponse | undefined>(
        getDevicesQueryKey(),
      );
      return findDeviceById(devices, deviceId);
    },
    refetchInterval,
  });
}

