import { Badge, Code, Group, Text, Tooltip } from "@mantine/core";
import type { DevicePairing } from "@/lib/api";

function formatInterval(seconds: number): string {
  if (seconds < 3600) return `${seconds / 60} min`;
  return `${seconds / 3600}h`;
}

interface Props {
  pairing: DevicePairing;
}

/** Compact horizontal summary of the config locked into a pairing at claim time. */
export function PairingConfigSummary({ pairing }: Props) {
  return (
    <Group gap="lg" wrap="wrap">
      <Group gap={6} align="center">
        <Text size="xs" c="dimmed">
          Server
        </Text>
        <Code style={{ fontSize: "var(--mantine-font-size-xs)" }}>
          {pairing.heartbeat_server_url}
        </Code>
      </Group>
      <Group gap={6} align="center">
        <Text size="xs" c="dimmed">
          Interval
        </Text>
        <Text size="xs">{formatInterval(pairing.interval_seconds)}</Text>
      </Group>
      <Group gap={6} align="center">
        <Text size="xs" c="dimmed">
          Biometric
        </Text>
        <Badge size="xs" color={pairing.app_biometric_enabled ? "teal" : "gray"} variant="light">
          {pairing.app_biometric_enabled ? "on" : "off"}
        </Badge>
      </Group>
      <Group gap={6} align="center">
        <Text size="xs" c="dimmed">
          Settings
        </Text>
        <Tooltip
          label={
            pairing.app_settings_locked
              ? "The user cannot change any settings in the companion app."
              : "The user can freely adjust settings in the companion app."
          }
          withArrow
          bg="dark.7"
          c="gray.1"
        >
          <Badge
            size="xs"
            color={pairing.app_settings_locked ? "orange" : "gray"}
            variant="light"
            style={{ cursor: "help" }}
          >
            {pairing.app_settings_locked ? "locked" : "user-editable"}
          </Badge>
        </Tooltip>
      </Group>
    </Group>
  );
}
