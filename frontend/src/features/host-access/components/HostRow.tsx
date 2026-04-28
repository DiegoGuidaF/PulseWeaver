import React from "react";
import { ActionIcon, Badge, Group, Table, Text, Tooltip } from "@mantine/core";
import { IconTrash } from "@tabler/icons-react";
import type { HostGroupWithMembers } from "@/lib/api";
import type { DraftHost, HostsDiff } from "@/features/host-access/drafts/knownHostsDraft";
import { resolveHostIcon } from "@/features/host-access/hostIconConfig";
import { GroupBadgeList } from "@/features/host-access/components/GroupBadgeList";

interface Props {
  draft: DraftHost;
  diff: HostsDiff;
  serverGroups: HostGroupWithMembers[];
  onIconClick: () => void;
  onDelete: () => void;
}

export function HostRow({ draft, diff, serverGroups, onIconClick, onDelete }: Props) {
  const resolved = resolveHostIcon(draft.icon);
  const isNew = typeof draft.id !== "number";
  const isIconChanged = diff.iconChanged.some((d) => d.id === draft.id);
  const isGroupsChanged = diff.groupsChanged.some((d) => d.id === draft.id);
  const dirty = isNew || isIconChanged || isGroupsChanged;

  const groupRefs = draft.groupIds
    .map((id) => serverGroups.find((g) => g.id === id))
    .filter((g): g is HostGroupWithMembers => g !== undefined)
    .map((g) => ({ id: g.id, name: g.name, icon: g.icon ?? null }));

  return (
    <Table.Tr>
      <Table.Td>
        <Group gap="xs" wrap="nowrap">
          <Tooltip label="Change icon" withArrow>
            <ActionIcon
              variant="subtle"
              size="sm"
              color="gray"
              onClick={onIconClick}
              aria-label={`Change icon for ${draft.fqdn}`}
            >
              {resolved.kind === "tabler" ? (
                React.createElement(resolved.icon, { size: 14, stroke: 1.5 })
              ) : (
                <Text size="sm" span>
                  {resolved.value}
                </Text>
              )}
            </ActionIcon>
          </Tooltip>
          <Text size="sm" fw={500} ff="monospace">
            {draft.fqdn}
          </Text>
          {isNew && draft.source === "suggestion" ? (
            <Badge size="xs" color="indigo" variant="light">
              From suggestion
            </Badge>
          ) : isNew ? (
            <Badge size="xs" color="teal" variant="light">
              New
            </Badge>
          ) : null}
          {dirty && !isNew && (
            <Badge size="xs" color="orange" variant="light">
              Edited
            </Badge>
          )}
        </Group>
      </Table.Td>
      <Table.Td>
        {groupRefs.length === 0 ? (
          <Text size="sm" c="dimmed">
            —
          </Text>
        ) : (
          <GroupBadgeList groups={groupRefs} />
        )}
      </Table.Td>
      <Table.Td>
        <UserCount draft={draft} />
      </Table.Td>
      <Table.Td>
        <Group gap={4} justify="flex-end">
          <Tooltip label="Stage delete" withArrow>
            <ActionIcon
              variant="subtle"
              color="red"
              size="sm"
              onClick={onDelete}
              aria-label={`Delete ${draft.fqdn}`}
            >
              <IconTrash size={14} stroke={1.5} />
            </ActionIcon>
          </Tooltip>
        </Group>
      </Table.Td>
    </Table.Tr>
  );
}

function UserCount({ draft }: { draft: DraftHost }) {
  const count = typeof draft.id === "number" ? null : null;
  return (
    <Text size="sm" c={count && count > 0 ? "indigo" : "dimmed"}>
      {count == null ? "—" : `${count} ${count === 1 ? "user" : "users"}`}
    </Text>
  );
}
