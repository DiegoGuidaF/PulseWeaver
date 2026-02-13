import { useQuery } from "@tanstack/react-query";
import { getDevicesOptions } from "@/lib/api/@tanstack/react-query.gen";

export function useDevices() {
  return useQuery(getDevicesOptions());
}
