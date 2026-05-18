import React from "react";
import { Link } from "react-router-dom";
import {
  ActionIcon,
  Anchor,
  Badge,
  Button,
  Divider,
  Group,
  Paper,
  SimpleGrid,
  Stack,
  Text,
  ThemeIcon,
  Title,
  Tooltip,
} from "@mantine/core";
import { IconArrowBackUp, IconPencil, IconTrash } from "@tabler/icons-react";
import type { GroupDetailWithUsers, Id } from "@/lib/api";
import type { DraftGroup, GroupsDiff } from "@/features/host-access/drafts/hostGroupsDraft";
import { GroupMembershipTables } from "@/features/host-access/components/GroupMembershipTables";
import { resolveHostIcon } from "@/features/host-access/hostIconConfig";

interface HostRef {
  id: Id;
  fqdn: string;
}

interface Props {
  group: DraftGroup | null;
  serverGroup: GroupDetailWithUsers | null;
  diff: GroupsDiff;
  hosts: HostRef[];
  isTombstoned?: boolean;
  onEdit: () => void;
  onDelete: () => void;
  onRestore: () => void;
  onToggleHost: (hostId: Id) => void;
}

export function GroupDetailPanel({
  group,
  serverGroup,
  diff,
  hosts,
  isTombstoned,
  onEdit,
  onDelete,
  onRestore,
  onToggleHost,
}: Props) {
  if (!group) {
    return (
      <Paper withBorder radius="md" p="xl" h="100%">
        <Stack align="center" justify="center" h="100%" gap="xs">
          <Text c="dimmed" size="sm">
            Select a group to manage its hosts, or create a new one.
          </Text>
        </Stack>
      </Paper>
    );
  }

  const color = group.color;
  const resolved = resolveHostIcon(group.icon);
  const inGroupIds = new Set<Id>(
    group.hostIds.filter((id): id is Id => typeof id === "number"),
  );
  const dirtyEntry = diff.byId.get(group.id);
  const isAdded = dirtyEntry === "added";

  return (
    <Paper withBorder radius="md" p="md" h="100%">
      <Stack gap="md" h="100%">
        <Group justify="space-between" align="flex-start" wrap="nowrap">
          <Group gap="sm" wrap="nowrap" style={{ minWidth: 0 }}>
            <ThemeIcon variant="light" color={color} size={48} radius="md">
              {resolved.kind === "tabler" ? (
                React.createElement(resolved.icon, { size: 24, stroke: 1.5 })
              ) : (
                <Text size="xl" span>
                  {resolved.value}
                </Text>
              )}
            </ThemeIcon>
            <Stack gap={2} style={{ minWidth: 0 }}>
              <Group gap="xs" wrap="nowrap">
                <Title order={3} style={{ wordBreak: "break-word" }}>
                  {group.name || "Unnamed group"}
                </Title>
                {isAdded && (
                  <Badge size="xs" color="teal" variant="light">
                    New
                  </Badge>
                )}
                {isTombstoned && (
                  <Badge size="xs" color="red" variant="light">
                    Marked for deletion
                  </Badge>
                )}
              </Group>
              <Text size="sm" c="dimmed">
                {group.hostIds.length}{" "}
                {group.hostIds.length === 1 ? "host" : "hosts"}
                {group.description ? ` · ${group.description}` : ""}
              </Text>
            </Stack>
          </Group>
          <Group gap={4}>
            {isTombstoned ? (
              <Tooltip label="Undo delete" withArrow>
                <ActionIcon variant="subtle" size="md" onClick={onRestore} aria-label="Restore">
                  <IconArrowBackUp size={16} stroke={1.5} />
                </ActionIcon>
              </Tooltip>
            ) : (
              <>
                <Tooltip label="Edit metadata" withArrow>
                  <ActionIcon variant="subtle" size="md" onClick={onEdit} aria-label="Edit">
                    <IconPencil size={16} stroke={1.5} />
                  </ActionIcon>
                </Tooltip>
                <Tooltip label="Delete group" withArrow>
                  <ActionIcon
                    variant="subtle"
                    color="red"
                    size="md"
                    onClick={onDelete}
                    aria-label="Delete"
                  >
                    <IconTrash size={16} stroke={1.5} />
                  </ActionIcon>
                </Tooltip>
              </>
            )}
          </Group>
        </Group>

        {isTombstoned ? (
          <Stack align="center" py="lg" gap="xs">
            <Text size="sm" c="dimmed">
              This group will be deleted on save.
            </Text>
            <Button variant="outline" size="xs" onClick={onRestore}>
              Restore
            </Button>
          </Stack>
        ) : (
          <GroupMembershipTables
            hosts={hosts}
            inGroupIds={inGroupIds}
            onToggle={onToggleHost}
          />
        )}

        {serverGroup && !isAdded && (
          <AccessPanel serverGroup={serverGroup} />
        )}
      </Stack>
    </Paper>
  );
}

function AccessPanel({ serverGroup }: { serverGroup: GroupDetailWithUsers }) {
  const users = serverGroup.users ?? [];
  const policies = serverGroup.network_policies;

  if (users.length === 0 && policies.length === 0) return null;

  return (
    <>
      <Divider />
      <Stack gap="xs">
        <Text size="sm" fw={600} c="dimmed">
          Access · read-only
        </Text>
        <SimpleGrid cols={2} spacing="md">
          <Stack gap="xs">
            <Text size="xs" fw={700}>
              Users · {users.length}
            </Text>
            {users.length === 0 ? (
              <Text size="xs" c="dimmed">None</Text>
            ) : (
              <>
                {users.slice(0, 3).map((u) => (
                  <Text key={u.id} size="xs">
                    {u.display_name}{" "}
                    <Text span c="dimmed" ff="monospace">{u.username}</Text>
                  </Text>
                ))}
                {users.length > 3 && (
                  <Text size="xs" c="dimmed">…and {users.length - 3} more</Text>
                )}
                <Anchor component={Link} to={`/access/users?group_id=${serverGroup.id}`} size="xs">
                  View users →
                </Anchor>
              </>
            )}
          </Stack>
          <Stack gap="xs">
            <Text size="xs" fw={700}>
              Network policies · {policies.length}
            </Text>
            {policies.length === 0 ? (
              <Text size="xs" c="dimmed">None</Text>
            ) : (
              <>
                {policies.slice(0, 3).map((p) => (
                  <Text key={p.id} size="xs">
                    {p.name}{" "}
                    <Text span c="dimmed" ff="monospace">{p.cidr}</Text>
                  </Text>
                ))}
                {policies.length > 3 && (
                  <Text size="xs" c="dimmed">…and {policies.length - 3} more</Text>
                )}
                <Anchor component={Link} to={`/access/network-policies?group_id=${serverGroup.id}`} size="xs">
                  View policies →
                </Anchor>
              </>
            )}
          </Stack>
        </SimpleGrid>
      </Stack>
    </>
  );
}
