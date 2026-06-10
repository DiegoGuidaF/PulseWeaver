import { Badge, Tooltip } from "@mantine/core";
import { IconShieldOff } from "@tabler/icons-react";

const BASE_TOOLTIP = "Bypasses the host allowlist — reaches every host, including future ones";

export interface AllHostsBypassPillProps {
  /** Render the admin-reach variant (PW-75 F8) instead of the standard subject pill. */
  admin?: boolean;
  size?: "xs" | "sm" | "md";
}

/**
 * Warning-coloured pill surfacing that a subject's `bypass_host_check` grants
 * reach to every host (including future ones), in place of the old terse
 * "bypass ✱" / "Bypass" badges that carried no explanation.
 *
 * The `admin` prop is reserved for the "All hosts (admin)" variant (PW-75 F8)
 * so the same primitive can grow without restructuring call sites.
 */
export function AllHostsBypassPill({ admin = false, size = "sm" }: AllHostsBypassPillProps) {
  const label = admin ? "All hosts (admin)" : "All hosts";
  const tooltip = admin ? `${BASE_TOOLTIP} — including admin-only hosts` : BASE_TOOLTIP;

  return (
    <Tooltip label={tooltip} withArrow multiline maw={280}>
      <Badge variant="light" color="yellow" size={size} leftSection={<IconShieldOff size={12} />}>
        {label}
      </Badge>
    </Tooltip>
  );
}
