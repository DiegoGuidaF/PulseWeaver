import dayjs from "dayjs";
import { Badge, Tooltip } from "@mantine/core";
import { IconClock, IconPlugConnected, IconPlugConnectedX, IconStack2 } from "@tabler/icons-react";
import type { DevicePairingSummary, DeviceRuleSummary } from "@/lib/api";
import { DevicePairingStatus, RuleType } from "@/lib/api";

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
  rules: DeviceRuleSummary[];
  pairing?: DevicePairingSummary | null;
  liveAddressCount: number;
  size?: "xs" | "sm";
}

export function RuleChips({ rules, pairing, liveAddressCount, size = "xs" }: Props) {
  if (pairing?.status === DevicePairingStatus.PENDING) {
    const label = formatPairingExpiry(pairing.expires_at);
    const tooltipLabel = `Pairing pending · ${label}`;
    return (
      <Tooltip label={tooltipLabel} withArrow>
        <Badge
          size={size}
          color="indigo"
          variant="light"
          aria-label={tooltipLabel}
          leftSection={<IconPlugConnected size={10} stroke={1.5} aria-hidden="true" />}
        >
          {label}
        </Badge>
      </Tooltip>
    );
  }

  if (pairing?.status === DevicePairingStatus.EXPIRED) {
    const expiredDaysAgo = dayjs().diff(dayjs(pairing.expires_at), "day");
    if (expiredDaysAgo < 7) {
      const tooltipLabel = "Pairing code expired — regenerate required";
      return (
        <Tooltip label={tooltipLabel} withArrow>
          <Badge
            size={size}
            color="red"
            variant="light"
            aria-label={tooltipLabel}
            leftSection={<IconPlugConnectedX size={10} stroke={1.5} aria-hidden="true" />}
          >
            expired
          </Badge>
        </Tooltip>
      );
    }
  }

  return (
    <>
      {rules
        .filter((r) => r.enabled)
        .map((r) => {
          if (r.type === RuleType.AUTO_EXPIRY && r.ttl_seconds != null) {
            const tooltipLabel = `Auto-expiry · TTL ${formatTtl(r.ttl_seconds)}`;
            return (
              <Tooltip key="auto_expiry" label={tooltipLabel} withArrow>
                <Badge
                  size={size}
                  color="teal"
                  variant="light"
                  aria-label={tooltipLabel}
                  leftSection={<IconClock size={10} stroke={1.5} aria-hidden="true" />}
                >
                  {formatTtl(r.ttl_seconds)}
                </Badge>
              </Tooltip>
            );
          }
          if (r.type === RuleType.MAX_ACTIVE && r.limit != null) {
            const atLimit = liveAddressCount >= r.limit;
            const tooltipLabel = atLimit
              ? `Max active IPs · at limit (${liveAddressCount}/${r.limit}) · next IP will evict oldest`
              : `Max active IPs · ${liveAddressCount} of ${r.limit}`;
            return (
              <Tooltip key="max_active" label={tooltipLabel} withArrow>
                <Badge
                  size={size}
                  color={atLimit ? "orange" : "teal"}
                  variant={atLimit ? "filled" : "light"}
                  aria-label={tooltipLabel}
                  leftSection={<IconStack2 size={10} stroke={1.5} aria-hidden="true" />}
                >
                  {liveAddressCount}/{r.limit}
                </Badge>
              </Tooltip>
            );
          }
          return null;
        })}
    </>
  );
}
