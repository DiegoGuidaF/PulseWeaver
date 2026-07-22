import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  getDeviceAddressesQueryKey,
  getDeviceAddressLeaseRuleQueryKey,
  putDeviceAddressLeaseRuleMutation,
} from "@/lib/api/@tanstack/react-query.gen";
import { invalidateOwnerFleet } from "../fleetCache";

export function usePutDeviceAddressLeaseRule(deviceId: number, ownerId: number) {
  const queryClient = useQueryClient();

  return useMutation({
    ...putDeviceAddressLeaseRuleMutation({ path: { device_id: deviceId } }),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: getDeviceAddressLeaseRuleQueryKey({ path: { device_id: deviceId } }),
      });
      queryClient.invalidateQueries({
        queryKey: getDeviceAddressesQueryKey({ path: { device_id: deviceId } }),
      });
      // The owner's fleet group carries the rule chips, and rule changes evict or
      // expire addresses, so the live-address counts and derived device states in
      // that same group move with them.
      invalidateOwnerFleet(queryClient, ownerId);
    },
  });
}
