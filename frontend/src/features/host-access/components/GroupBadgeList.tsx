import { Badge, Group, Tooltip } from "@mantine/core";
import { GroupBadge } from "@/features/host-access/components/GroupBadge";

const MAX_VISIBLE = 3;

interface GroupRef {
  id: number;
  name: string;
  color: string;
  icon?: string | null;
}

interface Props {
  groups: GroupRef[];
  size?: "sm" | "xs";
  /** When provided alongside `onGroupClick`, badges become filter toggles: every group is shown (no overflow capping) and rendered "filled" or "outline" to reflect membership in this set. */
  selected?: Set<number>;
  /** Makes each badge clickable, toggling that group's membership in the caller's filter set. */
  onGroupClick?: (groupId: number) => void;
}

export function GroupBadgeList({ groups, size = "sm", selected, onGroupClick }: Props) {
  if (groups.length === 0) return null;

  if (onGroupClick) {
    return (
      <Group gap={4} wrap="wrap">
        {groups.map((g) => (
          <GroupBadge
            key={g.id}
            group={g}
            size={size}
            variant={selected?.has(g.id) ? "filled" : "outline"}
            onClick={(e) => {
              e.stopPropagation();
              onGroupClick(g.id);
            }}
          />
        ))}
      </Group>
    );
  }

  const visible = groups.slice(0, MAX_VISIBLE);
  const overflow = groups.slice(MAX_VISIBLE);

  return (
    <Group gap={4} wrap="nowrap">
      {visible.map((g) => (
        <GroupBadge key={g.id} group={g} size={size} />
      ))}
      {overflow.length > 0 && (
        <Tooltip
          label={overflow.map((g) => g.name).join(", ")}
          withArrow
          multiline
          maw={240}
        >
          <Badge variant="outline" color="gray" size={size} style={{ cursor: "default" }}>
            +{overflow.length} more
          </Badge>
        </Tooltip>
      )}
    </Group>
  );
}
