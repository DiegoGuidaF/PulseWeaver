import type { CSSProperties, MouseEvent, ReactNode } from "react";
import { Badge, Tooltip } from "@mantine/core";
import { resolveGroupIcon } from "@/features/host-access/hostIconConfig";

const MAX_LABEL_LEN = 18;

/** Alpha suffix appended to a 6-digit hex color (#RRGGBBAA) for the badge fill — bolder than Mantine's default "light" tint. */
const FILL_ALPHA = "55";

function intensifiedFill(color?: string | null): CSSProperties | undefined {
  if (!color) return undefined;
  return { backgroundColor: `${color}${FILL_ALPHA}`, color };
}

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
  onClick?: (event: MouseEvent) => void;
}

export function GroupBadge({ group, size = "sm", variant = "light", rightSection, onClick }: Props) {
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
      onClick={onClick}
      style={{
        ...(variant === "light" ? intensifiedFill(group.color) : undefined),
        ...(onClick ? { cursor: "pointer" } : undefined),
      }}
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
