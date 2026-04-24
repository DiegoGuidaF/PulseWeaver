import { useState } from "react";
import {
  ActionIcon,
  Button,
  Card,
  Group,
  Modal,
  SimpleGrid,
  Stack,
  Text,
  Title,
  Tooltip,
} from "@mantine/core";
import { notifications } from "@mantine/notifications";
import { IconPencil, IconPlus, IconTrash } from "@tabler/icons-react";
import type { HostGroupWithMembers, KnownHostWithStats } from "@/lib/api";
import { useDeleteHostGroup } from "@/features/host-access/hooks/useDeleteHostGroup";
import { GroupFormModal } from "@/features/host-access/components/GroupFormModal";
import { groupColor } from "@/features/host-access/utils/groupColor";
import { getHostIcon } from "@/features/host-access/hostIconConfig";
import { toErrorMessage } from "@/lib/api-client";

interface Props {
  groups: HostGroupWithMembers[];
  hosts: KnownHostWithStats[];
}

export function HostGroupsTab({ groups, hosts }: Props) {
  const [createModalOpen, setCreateModalOpen] = useState(false);
  const [editTarget, setEditTarget] = useState<HostGroupWithMembers | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<HostGroupWithMembers | null>(null);

  const deleteHostGroup = useDeleteHostGroup();

  function handleConfirmDelete() {
    if (!deleteTarget) return;
    deleteHostGroup.mutate(
      { path: { group_id: deleteTarget.id } },
      {
        onSuccess: () =>
          notifications.show({ color: "green", message: `Group "${deleteTarget.name}" deleted` }),
        onError: (err) =>
          notifications.show({ color: "red", title: "Failed to delete group", message: toErrorMessage(err) }),
        onSettled: () => setDeleteTarget(null),
      },
    );
  }

  if (groups.length === 0) {
    return (
      <>
        <Card withBorder>
          <Stack gap="md" align="center" py="xl">
            <Text fz={48}>🗂</Text>
            <Title order={3}>No groups yet</Title>
            <Text c="dimmed" size="sm" maw={440} ta="center">
              Bundle related hosts so you can grant access in one click — "Media", "Photos",
              "Storage". Groups are a UX convenience, not an authz concept.
            </Text>
            <Button
              leftSection={<IconPlus size={16} />}
              onClick={() => setCreateModalOpen(true)}
            >
              New group
            </Button>
          </Stack>
        </Card>
        <GroupFormModal
          opened={createModalOpen}
          onClose={() => setCreateModalOpen(false)}
          hosts={hosts}
        />
      </>
    );
  }

  return (
    <>
      <Modal
        opened={deleteTarget !== null}
        onClose={() => setDeleteTarget(null)}
        title="Delete group?"
        closeOnClickOutside={false}
        closeOnEscape={false}
        withCloseButton={false}
      >
        <Text size="sm">
          Delete group{" "}
          <Text component="span" fw={600}>
            {deleteTarget?.name}
          </Text>
          ? This cannot be undone.
        </Text>
        <Group justify="flex-end" mt="md" gap="xs">
          <Button variant="outline" onClick={() => setDeleteTarget(null)}>
            Cancel
          </Button>
          <Button
            color="red"
            onClick={handleConfirmDelete}
            disabled={deleteHostGroup.isPending}
            loading={deleteHostGroup.isPending}
          >
            Delete
          </Button>
        </Group>
      </Modal>

      <GroupFormModal
        opened={createModalOpen}
        onClose={() => setCreateModalOpen(false)}
        hosts={hosts}
      />

      {editTarget && (
        <GroupFormModal
          key={editTarget.id}
          opened={editTarget !== null}
          onClose={() => setEditTarget(null)}
          hosts={hosts}
          editingGroup={editTarget}
        />
      )}

      <Group justify="flex-end" mb="sm">
        <Button
          size="xs"
          leftSection={<IconPlus size={14} />}
          onClick={() => setCreateModalOpen(true)}
        >
          New group
        </Button>
      </Group>

      <SimpleGrid cols={{ base: 1, sm: 2 }} spacing="sm">
        {groups.map((g) => {
          const color = groupColor(g.name);
          const GroupIcon = getHostIcon(g.icon);
          return (
            <Card key={g.id} withBorder padding="md">
              <Group justify="space-between" align="flex-start" mb="sm">
                <Group gap="sm">
                  <ActionIcon variant="light" color={color} size="lg" radius="md" aria-label="group">
                    <GroupIcon size={18} stroke={1.5} />
                  </ActionIcon>
                  <div>
                    <Text fw={600}>{g.name}</Text>
                    <Text size="xs" c="dimmed">
                      {g.hosts.length} {g.hosts.length === 1 ? "host" : "hosts"}
                      {g.description && ` · ${g.description}`}
                    </Text>
                  </div>
                </Group>
                <Group gap={4}>
                  <Tooltip label="Edit group" withArrow>
                    <ActionIcon
                      variant="subtle"
                      size="sm"
                      onClick={() => setEditTarget(g)}
                      aria-label={`Edit group ${g.name}`}
                    >
                      <IconPencil size={14} stroke={1.5} />
                    </ActionIcon>
                  </Tooltip>
                  <Tooltip label="Delete group" withArrow>
                    <ActionIcon
                      variant="subtle"
                      color="red"
                      size="sm"
                      onClick={() => setDeleteTarget(g)}
                      aria-label={`Delete group ${g.name}`}
                    >
                      <IconTrash size={14} stroke={1.5} />
                    </ActionIcon>
                  </Tooltip>
                </Group>
              </Group>

              <Stack gap={4}>
                {g.hosts.length === 0 ? (
                  <Text size="sm" c="dimmed">
                    No hosts in this group.
                  </Text>
                ) : (
                  g.hosts.map((h) => {
                    const HostIcon = getHostIcon(h.icon);
                    return (
                      <Group
                        key={h.id}
                        gap="xs"
                        px="xs"
                        py={6}
                        style={{
                          background: "var(--mantine-color-default-hover)",
                          borderRadius: 4,
                        }}
                      >
                        <HostIcon size={14} stroke={1.5} color="var(--mantine-color-dimmed)" />
                        <Text size="sm" ff="monospace" style={{ flex: 1 }}>
                          {h.fqdn}
                        </Text>
                      </Group>
                    );
                  })
                )}
              </Stack>
            </Card>
          );
        })}
      </SimpleGrid>
    </>
  );
}
