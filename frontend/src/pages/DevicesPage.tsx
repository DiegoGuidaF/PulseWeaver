import { Stack, Title, Text } from "@mantine/core";
import { CreateDeviceForm } from "@/features/devices/CreateDeviceForm";
import { DeviceList } from "@/features/devices/DeviceList";

export function DevicesPage() {
  return (
    <Stack maw={1024} gap="xl">
      <div>
        <Title order={1}>Devices</Title>
        <Text c="dimmed" size="sm">Manage your registered devices.</Text>
      </div>
      <CreateDeviceForm />
      <DeviceList />
    </Stack>
  );
}
