import { useState } from "react";
import { Link } from "react-router-dom";
import {
  ActionIcon,
  Anchor,
  Badge,
  Button,
  Group,
  Modal,
  SegmentedControl,
  Stack,
  Table,
  Text,
} from "@mantine/core";
import { useDisclosure } from "@mantine/hooks";
import { IconEye, IconX } from "@tabler/icons-react";
import { notifications } from "@mantine/notifications";
import { useListRegistrations } from "./hooks/useListRegistrations";
import { useDeleteRegistration } from "./hooks/useDeleteRegistration";
import { InviteDetailPanel } from "./InviteDetailPanel";
import {
  EXPIRING_SOON_MS,
  FILTER_TAB_OPTIONS,
  STATUS_BADGE,
} from "./constants";
import type { FilterTab } from "./constants";
import type { PendingRegistration } from "@/lib/api";
import { useDateFormatter } from "@/contexts/useDateTimePrefs";
import { toErrorMessage } from "@/lib/api-client";
import { useListUsers } from "@/features/auth/hooks/useListUsers";
import { useCurrentUser } from "@/features/auth/hooks/useCurrentUser";

function isExpiringSoon(row: PendingRegistration): boolean {
  return (
    row.status === "pending" &&
    new Date(row.expires_at).getTime() - Date.now() < EXPIRING_SOON_MS
  );
}

export function InviteList() {
  const [tab, setTab] = useState<FilterTab>("pending");
  const [viewRow, setViewRow] = useState<PendingRegistration | null>(null);
  const [deleteRow, setDeleteRow] = useState<PendingRegistration | null>(null);
  const [viewOpened, { open: openView, close: closeView }] = useDisclosure();
  const [deleteOpened, { open: openDelete, close: closeDelete }] =
    useDisclosure();

  const formatDateTime = useDateFormatter();
  const deleteMutation = useDeleteRegistration();
  const { data: currentUser } = useCurrentUser();
  const { data: users } = useListUsers();

  const ownerLabels = new Map(
    (users ?? []).map((u) => [
      u.id,
      u.id === currentUser?.id ? `${u.display_name} (you)` : u.display_name,
    ]),
  );

  // 'pending' and 'all' are the two server-side variants.
  // 'used' and 'expired' filter the 'all' result client-side,
  // keeping the cache to just two entries.
  const queryStatus = tab === "pending" ? "pending" : "all";
  const { data = [], isLoading } = useListRegistrations(queryStatus);

  const rows =
    tab === "used"
      ? data.filter((r) => r.status === "used")
      : tab === "expired"
        ? data.filter((r) => r.status === "expired")
        : data;

  function handleViewCode(row: PendingRegistration) {
    setViewRow(row);
    openView();
  }

  function handleInvalidate(row: PendingRegistration) {
    setDeleteRow(row);
    openDelete();
  }

  function confirmDelete() {
    if (!deleteRow) return;
    deleteMutation.mutate(
      { path: { registration_id: deleteRow.id } },
      {
        onSuccess: () => {
          notifications.show({ color: "green", message: "Invite invalidated" });
          closeDelete();
          setDeleteRow(null);
        },
        onError: (err) =>
          notifications.show({
            color: "red",
            title: "Failed to invalidate invite",
            message: toErrorMessage(err),
          }),
      },
    );
  }

  return (
    <>
      <Stack gap="sm">
        <SegmentedControl
          value={tab}
          onChange={(v) => setTab(v as FilterTab)}
          data={FILTER_TAB_OPTIONS}
        />

        {isLoading ? (
          <Text c="dimmed">Loading…</Text>
        ) : rows.length === 0 ? (
          <Text c="dimmed">
            {tab === "pending"
              ? "No pending invites. Click Create invite to add one."
              : "No invites found."}
          </Text>
        ) : (
          <Table highlightOnHover>
            <Table.Thead>
              <Table.Tr>
                <Table.Th>Device name</Table.Th>
                <Table.Th>Owner</Table.Th>
                <Table.Th>Created</Table.Th>
                <Table.Th>Expires</Table.Th>
                <Table.Th>Status</Table.Th>
                <Table.Th>Actions</Table.Th>
              </Table.Tr>
            </Table.Thead>
            <Table.Tbody>
              {rows.map((row) => (
                <Table.Tr key={row.id}>
                  <Table.Td>
                    {row.status === "used" && row.created_device_id != null ? (
                      <Anchor
                        component={Link}
                        to={`/devices/${row.created_device_id}`}
                      >
                        {row.device_name}
                      </Anchor>
                    ) : (
                      row.device_name
                    )}
                  </Table.Td>
                  <Table.Td>{ownerLabels.get(row.owner_id) ?? "Unknown"}</Table.Td>
                  <Table.Td>{formatDateTime(row.created_at)}</Table.Td>
                  <Table.Td c={isExpiringSoon(row) ? "orange" : undefined}>
                    {formatDateTime(row.expires_at)}
                  </Table.Td>
                  <Table.Td>
                    <Badge
                      color={STATUS_BADGE[row.status].color}
                      variant="light"
                    >
                      {STATUS_BADGE[row.status].label}
                    </Badge>
                  </Table.Td>
                  <Table.Td>
                    {row.status === "pending" ? (
                      <Group gap="xs">
                        <ActionIcon
                          variant="subtle"
                          size="sm"
                          aria-label="View code"
                          onClick={() => handleViewCode(row)}
                        >
                          <IconEye size={16} />
                        </ActionIcon>
                        <ActionIcon
                          variant="subtle"
                          color="red"
                          size="sm"
                          aria-label="Invalidate invite"
                          onClick={() => handleInvalidate(row)}
                        >
                          <IconX size={16} />
                        </ActionIcon>
                      </Group>
                    ) : (
                      <Text c="dimmed" size="sm">
                        —
                      </Text>
                    )}
                  </Table.Td>
                </Table.Tr>
              ))}
            </Table.Tbody>
          </Table>
        )}
      </Stack>

      <Modal
        opened={viewOpened}
        onClose={closeView}
        title="Registration code"
        size="md"
      >
        {viewRow && <InviteDetailPanel registration={viewRow} />}
      </Modal>

      <Modal
        opened={deleteOpened}
        onClose={closeDelete}
        title="Invalidate invite"
        size="sm"
      >
        <Stack gap="md">
          <Text>
            This will prevent anyone from using this code. This action cannot be
            undone.
          </Text>
          <Group justify="flex-end" gap="sm">
            <Button variant="outline" onClick={closeDelete}>
              Cancel
            </Button>
            <Button
              color="red"
              onClick={confirmDelete}
              loading={deleteMutation.isPending}
            >
              Invalidate
            </Button>
          </Group>
        </Stack>
      </Modal>
    </>
  );
}
