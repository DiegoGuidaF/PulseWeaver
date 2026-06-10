import { ActionIcon, Badge, Group, Table, Text, Tooltip } from "@mantine/core";
import { IconTrash } from "@tabler/icons-react";
import type { GroupDetailWithUsers } from "@/lib/api";
import type { DraftHost, HostsDiff } from "@/features/host-access/drafts/knownHostsDraft";
import { GroupBadgeList } from "@/features/host-access/components/GroupBadgeList";

interface Props {
  draft: DraftHost;
  diff: HostsDiff;
  serverGroups: GroupDetailWithUsers[];
  onGroupClick: (groupId: number) => void;
  onDelete: () => void;
}

export function HostRow({ draft, diff, serverGroups, onGroupClick, onDelete }: Props) {
  const isNew = typeof draft.id !== "number";
  const isGroupsChanged = diff.groupsChanged.some((d) => d.id === draft.id);
  const dirty = isNew || isGroupsChanged;
  const unassigned = draft.groupIds.length === 0;

  const groupRefs = draft.groupIds
    .map((id) => serverGroups.find((g) => g.id === id))
    .filter((g): g is GroupDetailWithUsers => g !== undefined)
    .map((g) => ({ id: g.id, name: g.name, color: g.color, icon: g.icon }));

  return (
    <Table.Tr>
      <Table.Td>
        <Group gap="xs" wrap="nowrap">
          <Text
            size="sm"
            fw={500}
            ff="monospace"
            c={unassigned && !isNew ? "dimmed" : undefined}
          >
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
        {unassigned ? (
          <Text size="sm" c="dimmed" fs="italic">
            Unassigned
          </Text>
        ) : (
          <ClickableGroupBadgeList groups={groupRefs} onGroupClick={onGroupClick} />
        )}
      </Table.Td>
      <Table.Td>
        <Group gap={4} justify="flex-end">
          <Tooltip label="Stage delete" withArrow>
            <ActionIcon
              variant="subtle"
              color="red"
              size="md"
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

interface ClickableGroupBadgeListProps {
  groups: { id: number; name: string; color: string; icon?: string | null }[];
  onGroupClick: (groupId: number) => void;
}

function ClickableGroupBadgeList({ groups, onGroupClick }: ClickableGroupBadgeListProps) {
  return (
    <Group gap={4} wrap="nowrap">
      {groups.map((g) => (
        <Tooltip key={g.id} label={`Filter by ${g.name}`} withArrow>
          <span
            style={{ display: "inline-flex", alignItems: "center", minHeight: 24, cursor: "pointer" }}
            onClick={() => onGroupClick(g.id)}
            role="button"
            tabIndex={0}
            onKeyDown={(e) => { if (e.key === "Enter" || e.key === " ") onGroupClick(g.id); }}
          >
            <GroupBadgeList groups={[g]} />
          </span>
        </Tooltip>
      ))}
    </Group>
  );
}
