import { useState } from "react";
import { Link } from "react-router-dom";
import {
  ActionIcon,
  Button,
  Card,
  Group,
  Modal,
  Skeleton,
  Stack,
  Table,
  Text,
  Title,
} from "@mantine/core";
import { notifications } from "@mantine/notifications";
import { IconTrash } from "@tabler/icons-react";
import { format } from "date-fns";
import type { Device } from "@/lib/api";
import { useDevices } from "@/features/devices/hooks/useDevices";
import { useDeleteDevice } from "@/features/devices/hooks/useDeleteDevice";
import { toErrorMessage } from "@/lib/api-client";

export function DeviceList() {
  const { data: devices, isLoading, error } = useDevices();
  const deleteDevice = useDeleteDevice();
  const [deviceToDelete, setDeviceToDelete] = useState<Device | null>(null);

  function handleConfirmDelete() {
    if (!deviceToDelete) return;
    deleteDevice.mutate(
      { path: { device_id: deviceToDelete.id } },
      {
        onSuccess: () => notifications.show({ color: "green", message: "Device deleted" }),
        onError: (err) =>
          notifications.show({ color: "red", title: "Error deleting device", message: toErrorMessage(err) }),
        onSettled: () => setDeviceToDelete(null),
      },
    );
  }

  if (error) {
    return (
      <Text c="red" p="md">Error: {toErrorMessage(error)}</Text>
    );
  }

  if (isLoading) {
    return (
      <Card withBorder>
        <Title order={3} mb="md">Devices</Title>
        <Stack gap="sm">
          <Group justify="space-between" pb="sm" style={{ borderBottom: "1px solid var(--mantine-color-default-border)" }}>
            <Skeleton height={16} width={100} />
            <Skeleton height={16} width={100} />
            <Skeleton height={16} width={80} />
            <Skeleton height={16} width={150} />
          </Group>
          {Array.from({ length: 5 }).map((_, i) => (
            <Group key={i} justify="space-between" py="sm" style={{ borderBottom: "1px solid var(--mantine-color-default-border)" }}>
              <Skeleton height={16} width={120} />
              <Skeleton height={16} width={80} />
              <Skeleton height={16} width={200} />
            </Group>
          ))}
        </Stack>
      </Card>
    );
  }

  return (
    <Card withBorder>
      <Title order={3} mb="md">Devices</Title>
      <Table>
        <Table.Thead>
          <Table.Tr>
            <Table.Th>Name</Table.Th>
            <Table.Th>ID</Table.Th>
            <Table.Th>Key prefix</Table.Th>
            <Table.Th>Created At</Table.Th>
            <Table.Th>Active addresses</Table.Th>
            <Table.Th style={{ textAlign: "right" }}>Actions</Table.Th>
          </Table.Tr>
        </Table.Thead>
        <Table.Tbody>
          {devices?.length === 0 ? (
            <Table.Tr>
              <Table.Td colSpan={6} style={{ height: 128, textAlign: "center" }}>
                <Stack align="center" justify="center" gap={8}>
                  <Text c="dimmed">No devices found.</Text>
                  <Text size="sm" c="gray">Add a device above to get started.</Text>
                </Stack>
              </Table.Td>
            </Table.Tr>
          ) : (
            devices?.map((device) => (
              <Table.Tr key={device.id}>
                <Table.Td fw={500}>{device.name}</Table.Td>
                <Table.Td ff="monospace" fz="xs">{device.id}</Table.Td>
                <Table.Td ff="monospace" fz="xs" c="dimmed">{device.api_key_prefix}</Table.Td>
                <Table.Td>{format(new Date(device.created_at), "PP p")}</Table.Td>
                <Table.Td>
                  {device.address_count === 0 ? (
                    <Text c="dimmed">0</Text>
                  ) : (
                    <Text fw={500}>{device.address_count}</Text>
                  )}
                </Table.Td>
                <Table.Td>
                  <Group justify="flex-end" gap="sm">
                    <Button
                      component={Link}
                      to={`/devices/${device.id}`}
                      variant="outline"
                      size="sm"
                    >
                      Manage
                    </Button>
                    <ActionIcon
                      variant="subtle"
                      color="gray"
                      aria-label={`Delete device ${device.name}`}
                      onClick={() => setDeviceToDelete(device)}
                      disabled={deleteDevice.isPending}
                    >
                      <IconTrash size={16} stroke={1.5} />
                    </ActionIcon>
                  </Group>
                </Table.Td>
              </Table.Tr>
            ))
          )}
        </Table.Tbody>
      </Table>

      <Modal
        opened={deviceToDelete !== null}
        onClose={() => setDeviceToDelete(null)}
        title="Delete device"
      >
        <Text size="sm">
          Delete device &quot;{deviceToDelete?.name}&quot;? It will be hidden from the list and cannot receive addresses.
        </Text>
        <Group justify="flex-end" mt="md" gap="sm">
          <Button type="button" variant="outline" onClick={() => setDeviceToDelete(null)}>
            Cancel
          </Button>
          <Button
            type="button"
            color="red"
            onClick={handleConfirmDelete}
            disabled={deleteDevice.isPending}
          >
            {deleteDevice.isPending ? "Deleting..." : "Delete"}
          </Button>
        </Group>
      </Modal>
    </Card>
  );
}
