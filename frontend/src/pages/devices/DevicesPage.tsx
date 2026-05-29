import { Stack, Title, Text } from "@mantine/core";
import { OwnerGroupList } from "@/features/devices/OwnerGroupList";

export function DevicesPage() {
  return (
    <Stack maw={1024} gap="xl">
      <div>
        <Title order={1}>Devices</Title>
        <Text c="dimmed" mt={4}>All devices, grouped by owner.</Text>
      </div>
      <OwnerGroupList />
    </Stack>
  );
}
