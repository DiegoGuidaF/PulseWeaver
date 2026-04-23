import { useState } from "react";
import {
  ActionIcon,
  Badge,
  Button,
  Card,
  Group,
  Modal,
  Select,
  Stack,
  Table,
  Text,
  TextInput,
  Title,
  Tooltip,
} from "@mantine/core";
import { notifications } from "@mantine/notifications";
import { IconPencil, IconPlus, IconSearch, IconTrash, IconUpload } from "@tabler/icons-react";
import type { HostGroupWithMembers, KnownHostWithStats } from "@/lib/api";
import { useDeleteKnownHost } from "@/features/host-access/hooks/useDeleteKnownHost";
import { AddHostModal } from "@/features/host-access/components/AddHostModal";
import { BulkImportModal } from "@/features/host-access/components/BulkImportModal";
import { EditHostIconModal } from "@/features/host-access/components/EditHostIconModal";
import { toErrorMessage } from "@/lib/api-client";

interface Props {
  hosts: KnownHostWithStats[];
  groups: HostGroupWithMembers[];
}

export function KnownHostsTab({ hosts, groups }: Props) {
  const [search, setSearch] = useState("");
  const [groupFilter, setGroupFilter] = useState<string | null>(null);
  const [addModalOpen, setAddModalOpen] = useState(false);
  const [bulkModalOpen, setBulkModalOpen] = useState(false);
  const [editIconTarget, setEditIconTarget] = useState<KnownHostWithStats | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<KnownHostWithStats | null>(null);

  const deleteKnownHost = useDeleteKnownHost();

  const filtered = hosts.filter((h) => {
    const matchesSearch = h.fqdn.toLowerCase().includes(search.toLowerCase());
    if (!matchesSearch) return false;
    if (groupFilter === "__ungrouped__") return h.groups.length === 0;
    if (groupFilter) return h.groups.some((g) => String(g.id) === groupFilter);
    return true;
  });

  const groupSelectData = [
    { value: "__ungrouped__", label: "No group" },
    ...groups.map((g) => ({ value: String(g.id), label: g.name })),
  ];

  function handleConfirmDelete() {
    if (!deleteTarget) return;
    deleteKnownHost.mutate(
      { path: { host_id: deleteTarget.id } },
      {
        onSuccess: () =>
          notifications.show({ color: "green", message: `${deleteTarget.fqdn} removed` }),
        onError: (err) =>
          notifications.show({ color: "red", title: "Failed to delete host", message: toErrorMessage(err) }),
        onSettled: () => setDeleteTarget(null),
      },
    );
  }

  if (hosts.length === 0) {
    return (
      <>
        <Card withBorder>
          <Stack gap="md" align="center" py="xl">
            <Text fz={48}>📡</Text>
            <Title order={3}>No known hosts yet</Title>
            <Text c="dimmed" size="sm" maw={440} ta="center">
              Add hosts one at a time, or paste a newline-separated list to bootstrap your
              allowlist.
            </Text>
            <Group gap="xs">
              <Button leftSection={<IconPlus size={16} />} onClick={() => setAddModalOpen(true)}>
                Add host
              </Button>
              <Button
                variant="outline"
                leftSection={<IconUpload size={16} />}
                onClick={() => setBulkModalOpen(true)}
              >
                Bulk import
              </Button>
            </Group>
          </Stack>
        </Card>
        <AddHostModal opened={addModalOpen} onClose={() => setAddModalOpen(false)} />
        <BulkImportModal
          opened={bulkModalOpen}
          onClose={() => setBulkModalOpen(false)}
          existingFqdns={hosts.map((h) => h.fqdn)}
        />
      </>
    );
  }

  return (
    <>
      <Modal
        opened={deleteTarget !== null}
        onClose={() => setDeleteTarget(null)}
        title="Delete host?"
        closeOnClickOutside={false}
        closeOnEscape={false}
        withCloseButton={false}
      >
        <Text size="sm">
          Remove{" "}
          <Text component="span" fw={600} ff="monospace">
            {deleteTarget?.fqdn}
          </Text>{" "}
          from known hosts?
        </Text>
        {deleteTarget && deleteTarget.user_count > 0 && (
          <Text size="sm" c="dimmed" mt="xs">
            Currently granted to{" "}
            <Text component="span" fw={600}>
              {deleteTarget.user_count} {deleteTarget.user_count === 1 ? "user" : "users"}
            </Text>
            . Those grants will be removed.
          </Text>
        )}
        <Group justify="flex-end" mt="md" gap="xs">
          <Button variant="outline" onClick={() => setDeleteTarget(null)}>
            Cancel
          </Button>
          <Button
            color="red"
            onClick={handleConfirmDelete}
            disabled={deleteKnownHost.isPending}
            loading={deleteKnownHost.isPending}
          >
            Delete
          </Button>
        </Group>
      </Modal>

      <AddHostModal opened={addModalOpen} onClose={() => setAddModalOpen(false)} />
      <BulkImportModal
        opened={bulkModalOpen}
        onClose={() => setBulkModalOpen(false)}
        existingFqdns={hosts.map((h) => h.fqdn)}
      />
      {editIconTarget && (
        <EditHostIconModal
          key={editIconTarget.id}
          opened={editIconTarget !== null}
          onClose={() => setEditIconTarget(null)}
          hostId={editIconTarget.id}
          hostFqdn={editIconTarget.fqdn}
          currentIcon={editIconTarget.icon ?? null}
        />
      )}

      <Card withBorder>
        <Group justify="space-between" mb="sm">
          <Group gap="xs">
            <TextInput
              placeholder="Search hosts…"
              value={search}
              onChange={(e) => setSearch(e.currentTarget.value)}
              leftSection={<IconSearch size={14} />}
              w={240}
            />
            <Select
              placeholder="All groups"
              data={groupSelectData}
              value={groupFilter}
              onChange={setGroupFilter}
              clearable
              w={180}
            />
          </Group>
          <Group gap="xs">
            <Button
              variant="subtle"
              size="xs"
              leftSection={<IconUpload size={14} />}
              onClick={() => setBulkModalOpen(true)}
            >
              Bulk import
            </Button>
            <Button
              size="xs"
              leftSection={<IconPlus size={14} />}
              onClick={() => setAddModalOpen(true)}
            >
              New host
            </Button>
          </Group>
        </Group>

        <Table.ScrollContainer minWidth={500}>
          <Table>
            <Table.Thead>
              <Table.Tr>
                <Table.Th>FQDN</Table.Th>
                <Table.Th>Groups</Table.Th>
                <Table.Th>Users with access</Table.Th>
                <Table.Th />
              </Table.Tr>
            </Table.Thead>
            <Table.Tbody>
              {filtered.length === 0 ? (
                <Table.Tr>
                  <Table.Td colSpan={4}>
                    <Text size="sm" c="dimmed" ta="center" py="md">
                      No hosts match the current filter.
                    </Text>
                  </Table.Td>
                </Table.Tr>
              ) : (
                filtered.map((h) => (
                  <Table.Tr key={h.id}>
                    <Table.Td>
                      <Text size="sm" fw={500} ff="monospace">
                        {h.fqdn}
                      </Text>
                    </Table.Td>
                    <Table.Td>
                      {h.groups.length === 0 ? (
                        <Text size="sm" c="dimmed">
                          —
                        </Text>
                      ) : (
                        <Group gap="xs">
                          {h.groups.map((g) => (
                            <Badge key={g.id} variant="light" color="indigo" size="sm">
                              {g.name}
                            </Badge>
                          ))}
                        </Group>
                      )}
                    </Table.Td>
                    <Table.Td>
                      <Text
                        size="sm"
                        c={h.user_count === 0 ? "dimmed" : "indigo"}
                        fw={h.user_count > 0 ? 500 : 400}
                      >
                        {h.user_count} {h.user_count === 1 ? "user" : "users"}
                      </Text>
                    </Table.Td>
                    <Table.Td>
                      <Group gap={4} justify="flex-end">
                        <Tooltip label="Edit icon" withArrow>
                          <ActionIcon
                            variant="subtle"
                            size="sm"
                            onClick={() => setEditIconTarget(h)}
                            aria-label={`Edit icon for ${h.fqdn}`}
                          >
                            <IconPencil size={14} stroke={1.5} />
                          </ActionIcon>
                        </Tooltip>
                        <Tooltip label="Delete host" withArrow>
                          <ActionIcon
                            variant="subtle"
                            color="red"
                            size="sm"
                            onClick={() => setDeleteTarget(h)}
                            aria-label={`Delete ${h.fqdn}`}
                          >
                            <IconTrash size={14} stroke={1.5} />
                          </ActionIcon>
                        </Tooltip>
                      </Group>
                    </Table.Td>
                  </Table.Tr>
                ))
              )}
            </Table.Tbody>
          </Table>
        </Table.ScrollContainer>
      </Card>
    </>
  );
}
