import type { ReactNode } from "react";
import { Badge, Tooltip } from "@mantine/core";
import { resolveGroupIcon } from "@/features/host-access/hostIconConfig";

const MAX_LABEL_LEN = 18;

interface GroupRef {
  id: number;
  name: string;
  color?: string | null;
  icon?: string | null;
}

interface Props {
  group: GroupRef;
  size?: "sm" | "xs";
  variant?: "light" | "filled" | "outline";
  rightSection?: ReactNode;
}

export function GroupBadge({ group, size = "sm", variant = "light", rightSection }: Props) {
  const truncated = group.name.length > MAX_LABEL_LEN
    ? group.name.slice(0, MAX_LABEL_LEN) + "…"
    : group.name;
  const needsTooltip = group.name.length > MAX_LABEL_LEN;

  const badge = (
    <Badge
      variant={variant}
      color={group.color ?? "gray"}
      size={size}
      leftSection={resolveGroupIcon(group.icon)({ size: 10 })}
      rightSection={rightSection}
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
