import { Stack } from "@mantine/core";
import { AddressLeaseRuleCard } from "@/features/devices/AddressLeaseRuleCard";
import { MaxActiveIpsRuleCard } from "@/features/devices/MaxActiveIpsRuleCard";

interface DeviceRulesTabProps {
  deviceId: number;
  liveAddressCount: number;
}

export function DeviceRulesTab({ deviceId, liveAddressCount }: DeviceRulesTabProps) {
  return (
    <Stack gap="sm">
      <AddressLeaseRuleCard deviceId={deviceId} />
      <MaxActiveIpsRuleCard deviceId={deviceId} liveAddressCount={liveAddressCount} />
    </Stack>
  );
}
