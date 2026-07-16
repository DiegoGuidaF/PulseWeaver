import { useQuery } from "@tanstack/react-query";
import { getDeviceAddressLeaseRuleOptions } from "@/lib/api/@tanstack/react-query.gen";

export function useDeviceAddressLeaseRule(deviceId: number) {
  return useQuery(getDeviceAddressLeaseRuleOptions({ path: { device_id: deviceId } }));
}
