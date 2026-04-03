import { useState } from "react";
import { useNavigate } from "react-router-dom";
import {
  ActionIcon,
  Button,
  Card,
  Chip,
  Group,
  Modal,
  Select,
  Skeleton,
  Stack,
  Table,
  Text,
  Title,
  Tooltip,
} from "@mantine/core";
import { notifications } from "@mantine/notifications";
import { IconTrash } from "@tabler/icons-react";
import { useDateFormatter } from "@/contexts/useDateTimePrefs";
import type { Device } from "@/lib/api";
import { UserRole } from "@/lib/api";
import { useCurrentUser } from "@/features/auth/hooks/useCurrentUser";
import { useListUsers } from "@/features/auth/hooks/useListUsers";
import { useDevices } from "@/features/devices/hooks/useDevices";
import { useGetDevicesByUser } from "@/features/devices/hooks/useGetDevicesByUser";
import { useDeleteDevice } from "@/features/devices/hooks/useDeleteDevice";
import { toErrorMessage } from "@/lib/api-client";

const CHIP_FILTER_MAX_USERS = 8;

export function DeviceList() {
  const navigate = useNavigate();
  const formatDateTime = useDateFormatter();
  const { data: currentUser } = useCurrentUser();
  const isAdmin = currentUser?.role === UserRole.ADMIN;

  const { data: users } = useListUsers({ enabled: isAdmin });
  const [ownerFilter, setOwnerFilter] = useState<number | null>(null);

  // Both queries run unconditionally to respect Rules of Hooks;
  // only the relevant result is used based on ownerFilter.
  const allDevices = useDevices();
  const userDevices = useGetDevicesByUser(ownerFilter ?? 0);

  const { data: devices, isLoading, error } =
    ownerFilter !== null ? userDevices : allDevices;

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

  // Show Owner column only when admin and no specific owner is filtered.
  const showOwnerCol = isAdmin && ownerFilter === null;

  // Chip filter: show chips for ≤8 users, fall back to a Select for larger lists.
  const userList = users ?? [];
  const useChips = userList.length <= CHIP_FILTER_MAX_USERS;

  function renderOwnerFilter() {
    if (!isAdmin || userList.length === 0) return null;

    if (useChips) {
      return (
        <Chip.Group
          value={ownerFilter !== null ? String(ownerFilter) : ""}
          onChange={(val) => setOwnerFilter(val ? Number(val) : null)}
        >
          <Group gap="xs" mb="sm">
            <Chip value="">All</Chip>
            {userList.map((u) => (
              <Chip key={u.id} value={String(u.id)}>
                {u.display_name}
              </Chip>
            ))}
          </Group>
        </Chip.Group>
      );
    }

    return (
      <Select
        placeholder="All owners"
        clearable
        searchable
        data={userList.map((u) => ({ value: String(u.id), label: u.display_name }))}
        value={ownerFilter !== null ? String(ownerFilter) : null}
        onChange={(val) => setOwnerFilter(val ? Number(val) : null)}
        mb="sm"
        style={{ maxWidth: 240 }}
      />
    );
  }

  if (error) {
    return (
      <Text c="red" p="md">Error: {toErrorMessage(error)}</Text>
    );
  }

  if (isLoading) {
    const colCount = showOwnerCol ? 6 : 5;
    return (
      <Card withBorder>
        <Title order={3} mb="md">Devices</Title>
        <Stack gap="sm">
          <Group justify="space-between" pb="sm" style={{ borderBottom: "1px solid var(--mantine-color-default-border)" }}>
            {Array.from({ length: colCount }).map((_, i) => (
              <Skeleton key={i} height={16} width={100} />
            ))}
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

  const colSpan = showOwnerCol ? 6 : 5;
  const emptyMessage =
    ownerFilter !== null
      ? "No devices for this owner."
      : "No devices found.";

  return (
    <Card withBorder>
      <Title order={3} mb="md">Devices</Title>
      {renderOwnerFilter()}
      <Table highlightOnHover>
        <Table.Thead>
          <Table.Tr>
            <Table.Th>Name</Table.Th>
            <Table.Th>Key prefix</Table.Th>
            <Table.Th>Created</Table.Th>
            <Table.Th>Live IPs</Table.Th>
            {showOwnerCol && <Table.Th>Owner</Table.Th>}
            <Table.Th w={48} />
          </Table.Tr>
        </Table.Thead>
        <Table.Tbody>
          {devices?.length === 0 ? (
            <Table.Tr>
              <Table.Td colSpan={colSpan} style={{ height: 128, textAlign: "center" }}>
                <Stack align="center" justify="center" gap={8}>
                  <Text c="dimmed">{emptyMessage}</Text>
                  {ownerFilter === null && (
                    <Text size="sm" c="gray">Add a device above to get started.</Text>
                  )}
                </Stack>
              </Table.Td>
            </Table.Tr>
          ) : (
            devices?.map((device) => (
              <Table.Tr
                key={device.id}
                style={{ cursor: "pointer" }}
                onClick={() => navigate(`/devices/${device.id}`)}
              >
                <Table.Td fw={500}>{device.name}</Table.Td>
                <Table.Td ff="monospace" fz="xs" c="dimmed">{device.api_key_prefix}</Table.Td>
                <Table.Td>{formatDateTime(device.created_at)}</Table.Td>
                <Table.Td>
                  {device.address_count === 0 ? (
                    <Text c="dimmed">0</Text>
                  ) : (
                    <Text fw={500} c="orange.4">{device.address_count}</Text>
                  )}
                </Table.Td>
                {showOwnerCol && (
                  <Table.Td c="dimmed">{device.owner_name ?? "—"}</Table.Td>
                )}
                <Table.Td>
                  <Tooltip label="Delete device" withArrow>
                    <ActionIcon
                      variant="subtle"
                      color="red"
                      aria-label={`Delete device ${device.name}`}
                      onClick={(e) => {
                        e.stopPropagation();
                        setDeviceToDelete(device);
                      }}
                      disabled={deleteDevice.isPending}
                    >
                      <IconTrash size={16} stroke={1.5} />
                    </ActionIcon>
                  </Tooltip>
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
