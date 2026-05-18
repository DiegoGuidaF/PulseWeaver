import { Badge, Group, Tooltip } from "@mantine/core";
import { GroupBadge } from "@/features/host-access/components/GroupBadge";

const MAX_VISIBLE = 2;

interface GroupRef {
  id: number;
  name: string;
  color: string;
  icon?: string | null;
}

interface Props {
  groups: GroupRef[];
  size?: "sm" | "xs";
}

export function GroupBadgeList({ groups, size = "sm" }: Props) {
  if (groups.length === 0) return null;

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
