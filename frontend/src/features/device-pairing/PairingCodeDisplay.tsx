import dayjs from "dayjs";
import {
  Badge,
  Button,
  Code,
  Divider,
  Group,
  List,
  Stack,
  Text,
  Tooltip,
} from "@mantine/core";
import { IconAlertTriangle, IconCopy, IconTrash } from "@tabler/icons-react";
import { notifications } from "@mantine/notifications";
import type { DevicePairing } from "@/lib/api";
import { toErrorMessage } from "@/lib/api-client";
import { useClipboard } from "@/hooks/useClipboard";
import { useDeleteDevicePairing } from "./hooks/useDeleteDevicePairing";

function formatTtl(expiresAt: string): string {
  const diffMin = dayjs(expiresAt).diff(dayjs(), "minute");
  if (diffMin <= 0) return "expired";
  if (diffMin < 60) return `${diffMin}m remaining`;
  const h = Math.floor(diffMin / 60);
  const m = diffMin % 60;
  return m > 0 ? `${h}h ${m}m remaining` : `${h}h remaining`;
}

function formatInterval(seconds: number): string {
  if (seconds < 3600) return `${seconds / 60} min`;
  return `${seconds / 3600}h`;
}

interface Props {
  deviceId: number;
  pairing: DevicePairing;
  onRevoke: () => void;
}

export function PairingCodeDisplay({ deviceId, pairing, onRevoke }: Props) {
  const { copy } = useClipboard();
  const deleteMutation = useDeleteDevicePairing(deviceId);

  function handleRevoke() {
    deleteMutation.mutate(
      { path: { id: deviceId, pairingId: pairing.id } },
      {
        onSuccess: () => {
          notifications.show({ color: "green", message: "Pairing code revoked" });
          onRevoke();
        },
        onError: (err) =>
          notifications.show({
            color: "red",
            title: "Failed to revoke pairing code",
            message: toErrorMessage(err),
          }),
      },
    );
  }

  return (
    <Stack gap="md">
      {/* 1. The code — primary focus */}
      <div>
        <Text size="sm" c="dimmed" mb={6}>
          Share this code with the end user
        </Text>
        <Code
          block
          style={{
            fontSize: 14,
            fontWeight: 600,
            padding: "10px 16px",
            wordBreak: "break-all",
          }}
        >
          {pairing.pairing_code}
        </Code>
      </div>

      {/* 2. Actions + TTL */}
      <Group justify="space-between" align="center">
        <Group gap="sm">
          <Button
            variant="default"
            size="sm"
            leftSection={<IconCopy size={14} />}
            onClick={() => copy(pairing.pairing_code, { successMessage: "Pairing code copied" })}
          >
            Copy code
          </Button>
          <Text size="sm" c="dimmed">
            {formatTtl(pairing.expires_at)}
          </Text>
        </Group>
        <Button
          variant="light"
          color="red"
          size="sm"
          leftSection={<IconTrash size={14} />}
          onClick={handleRevoke}
          loading={deleteMutation.isPending}
        >
          Revoke
        </Button>
      </Group>

      <Divider />

      {/* 3. Config summary — compact horizontal row */}
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
          <Badge
            size="xs"
            color={pairing.app_biometric_enabled ? "teal" : "gray"}
            variant="light"
          >
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

      <Divider />

      {/* 4. Instructions — secondary */}
      <Stack gap="xs">
        <Text size="sm" fw={500}>
          What the end user does
        </Text>
        <List size="sm" spacing="xs">
          <List.Item>Install the Heartbeat client companion app.</List.Item>
          <List.Item>
            On first launch, paste this code and tap <strong>Pair</strong>.
          </List.Item>
          <List.Item>Done — the app heartbeats and PulseWeaver picks up their IP.</List.Item>
        </List>
      </Stack>

      {/* 5. Warning — inline, not an alert box */}
      <Group gap={6} align="flex-start" wrap="nowrap">
        <IconAlertTriangle
          size={13}
          style={{ color: "var(--mantine-color-orange-5)", flexShrink: 0, marginTop: 2 }}
        />
        <Text size="xs" c="dimmed" style={{ lineHeight: 1.5 }}>
          When the user claims this code, the device's current API key is revoked and replaced.
          Anything using the old key — scripts, a previous companion install — will stop working.
        </Text>
      </Group>
    </Stack>
  );
}
