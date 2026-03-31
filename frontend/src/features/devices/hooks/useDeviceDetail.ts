import { useDevice } from "./useDevice";

export function useDeviceDetail(deviceId: number, refetchInterval: number | false = false) {
  return useDevice(deviceId, refetchInterval);
}
