import { useQuery } from "@tanstack/react-query";
import { getMaxActiveAddressesRuleOptions } from "@/lib/api/@tanstack/react-query.gen";

export function useMaxActiveAddressesRule(deviceId: number) {
  return useQuery(getMaxActiveAddressesRuleOptions({ path: { device_id: deviceId } }));
}
