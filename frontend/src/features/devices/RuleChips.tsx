import dayjs from "dayjs";
import { Badge, Tooltip } from "@mantine/core";
import { IconClock, IconStack2 } from "@tabler/icons-react";
import type { DeviceListEntry, DeviceRuleSummary } from "@/lib/api";
import { DeviceState } from "@/lib/api";

function formatTtl(seconds: number): string {
  if (seconds < 60) return `${seconds}s`;
  if (seconds < 3600) return `${Math.round(seconds / 60)}m`;
  return `${Math.round(seconds / 3600)}h`;
}

function formatPairingExpiry(expiresAt: string): string {
  const diffMin = dayjs(expiresAt).diff(dayjs(), "minute");
  if (diffMin <= 0) return "expired";
  if (diffMin < 60) return `${diffMin}m left`;
  return `${Math.floor(diffMin / 60)}h left`;
}

interface Props {
  entry: DeviceListEntry;
  size?: "xs" | "sm";
}

export function RuleChips({ entry, size = "xs" }: Props) {
  if (entry.state === DeviceState.PENDING_CLAIM && entry.pairing) {
    const label = formatPairingExpiry(entry.pairing.expires_at);
    return (
      <Tooltip label={`Pairing pending · ${label}`} withArrow>
        <Badge
          size={size}
          color="indigo"
          variant="light"
          leftSection={<IconClock size={10} stroke={1.5} />}
        >
          {label}
        </Badge>
      </Tooltip>
    );
  }

  return (
    <>
      {entry.rules
        .filter((r: DeviceRuleSummary) => r.enabled)
        .map((r: DeviceRuleSummary) => {
          if (r.type === "auto_expiry" && r.ttl_seconds != null) {
            return (
              <Tooltip
                key="auto_expiry"
                label={`Auto-expiry · TTL ${formatTtl(r.ttl_seconds)}`}
                withArrow
              >
                <Badge
                  size={size}
                  color="teal"
                  variant="light"
                  leftSection={<IconClock size={10} stroke={1.5} />}
                >
                  {formatTtl(r.ttl_seconds)}
                </Badge>
              </Tooltip>
            );
          }
          if (r.type === "max_active" && r.limit != null) {
            const current = entry.live_address_count;
            const atLimit = current >= r.limit;
            const tooltipLabel = atLimit
              ? `Max active IPs · at limit (${current}/${r.limit}) · next IP will evict oldest`
              : `Max active IPs · ${current} of ${r.limit}`;
            return (
              <Tooltip key="max_active" label={tooltipLabel} withArrow>
                <Badge
                  size={size}
                  color={atLimit ? "orange" : "teal"}
                  variant={atLimit ? "filled" : "light"}
                  leftSection={<IconStack2 size={10} stroke={1.5} />}
                >
                  {current}/{r.limit}
                </Badge>
              </Tooltip>
            );
          }
          return null;
        })}
    </>
  );
}
