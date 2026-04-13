import { Stack } from "@mantine/core";
import { AddressLeaseRuleCard } from "@/features/devices/AddressLeaseRuleCard";
import { MaxActiveIpsRuleCard } from "@/features/devices/MaxActiveIpsRuleCard";

interface DeviceRulesTabProps {
  deviceId: number;
}

export function DeviceRulesTab({ deviceId }: DeviceRulesTabProps) {
  return (
    <Stack gap="sm">
      <AddressLeaseRuleCard deviceId={deviceId} />
      <MaxActiveIpsRuleCard deviceId={deviceId} />
    </Stack>
  );
}
