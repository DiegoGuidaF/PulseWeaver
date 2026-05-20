import { Stack, Title, Text } from "@mantine/core";
import { DeviceOwnerGroupList } from "@/features/devices/DeviceOwnerGroupList";

export function UserDevicesPage() {
  return (
    <Stack maw={1024} gap="xl">
      <div>
        <Title order={1}>Devices</Title>
        <Text c="dimmed" size="sm">All devices, grouped by owner.</Text>
      </div>
      <DeviceOwnerGroupList />
    </Stack>
  );
}
