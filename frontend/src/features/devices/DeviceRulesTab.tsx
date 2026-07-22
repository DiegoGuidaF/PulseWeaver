import { Stack } from "@mantine/core";
import { AddressLeaseRuleCard } from "@/features/devices/AddressLeaseRuleCard";
import { MaxActiveIpsRuleCard } from "@/features/devices/MaxActiveIpsRuleCard";

interface DeviceRulesTabProps {
  deviceId: number;
  ownerId: number;
}

export function DeviceRulesTab({ deviceId, ownerId }: DeviceRulesTabProps) {
  return (
    <Stack gap="sm">
      <AddressLeaseRuleCard deviceId={deviceId} ownerId={ownerId} />
      <MaxActiveIpsRuleCard deviceId={deviceId} ownerId={ownerId} />
    </Stack>
  );
}
