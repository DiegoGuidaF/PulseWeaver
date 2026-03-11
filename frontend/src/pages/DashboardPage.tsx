import { Stack, Title, Text } from "@mantine/core";
import { CreateDeviceForm } from "@/features/devices/CreateDeviceForm";
import { DeviceList } from "@/features/devices/DeviceList";

export function DashboardPage() {
  return (
    <Stack maw={1024} gap="xl">
      <div>
        <Title order={1}>WallyDic Manager</Title>
        <Text c="dimmed">Manage your networked devices and addresses.</Text>
      </div>
      <Stack gap="xl">
        <CreateDeviceForm />
        <DeviceList />
      </Stack>
    </Stack>
  );
}
