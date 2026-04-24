import React from "react";
import { Badge, Tooltip } from "@mantine/core";
import { groupColor } from "@/features/host-access/utils/groupColor";
import { getHostIcon } from "@/features/host-access/hostIconConfig";

const MAX_LABEL_LEN = 18;

interface GroupRef {
  id: number;
  name: string;
  icon?: string | null;
}

interface Props {
  group: GroupRef;
  size?: "sm" | "xs";
}

export function GroupBadge({ group, size = "sm" }: Props) {
  const truncated = group.name.length > MAX_LABEL_LEN
    ? group.name.slice(0, MAX_LABEL_LEN) + "…"
    : group.name;
  const needsTooltip = group.name.length > MAX_LABEL_LEN;

  const badge = (
    <Badge
      variant="light"
      color={groupColor(group.name)}
      size={size}
      leftSection={React.createElement(getHostIcon(group.icon), { size: 10, stroke: 1.5 })}
    >
      {truncated}
    </Badge>
  );

  if (needsTooltip) {
    return (
      <Tooltip label={group.name} withArrow>
        {badge}
      </Tooltip>
    );
  }
  return badge;
}
