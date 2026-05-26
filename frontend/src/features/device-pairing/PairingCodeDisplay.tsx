import dayjs from "dayjs";
import {
  Alert,
  Badge,
  Box,
  Button,
  Code,
  Divider,
  Group,
  List,
  Stack,
  Text,
  Title,
} from "@mantine/core";
import { IconAlertTriangle, IconCopy } from "@tabler/icons-react";
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
    <Group align="flex-start" gap="xl" wrap="nowrap">
      {/* Left column */}
      <Stack style={{ flex: 1, minWidth: 0 }}>
        <div>
          <Text size="sm" c="dimmed" mb="xs">
            Share this code with the end user
          </Text>
          <Group gap="sm" align="center">
            <Code
              style={{
                fontSize: 28,
                fontWeight: 700,
                letterSpacing: "0.15em",
                padding: "8px 16px",
              }}
            >
              {pairing.pairing_code}
            </Code>
            <Button
              variant="default"
              size="sm"
              leftSection={<IconCopy size={14} />}
              onClick={() => copy(pairing.pairing_code, { successMessage: "Pairing code copied" })}
            >
              Copy code
            </Button>
          </Group>
          <Text size="sm" c="dimmed" mt="xs">
            {formatTtl(pairing.expires_at)}
          </Text>
        </div>

        <Divider />

        <div>
          <Text size="sm" fw={500} mb="xs">
            What the end user does
          </Text>
          <List size="sm" spacing="xs">
            <List.Item>Install the Heartbeat client companion app.</List.Item>
            <List.Item>On first launch, paste this code and tap <strong>Pair</strong>.</List.Item>
            <List.Item>Done — the app heartbeats and PulseWeaver picks up their IP.</List.Item>
          </List>
        </div>

        <Alert
          color="orange"
          icon={<IconAlertTriangle size={16} />}
          title="Heads-up"
        >
          When the user claims this code, the device's current API key will be
          revoked and replaced with a new one issued to the companion. Anything
          still using the old key — scripts, a previous companion install — will
          stop working.
        </Alert>
      </Stack>

      {/* Right column */}
      <Box style={{ width: 220, flexShrink: 0 }}>
        <Stack gap="sm">
          <div>
            <Title order={6} mb="xs">
              App will be configured
            </Title>
            <Stack gap={4}>
              <Text size="sm">
                <Text span c="dimmed">Server: </Text>
                <Code>{pairing.heartbeat_server_url}</Code>
              </Text>
              <Text size="sm">
                <Text span c="dimmed">Interval: </Text>
                {formatInterval(pairing.interval_seconds)}
              </Text>
              <Text size="sm">
                <Text span c="dimmed">Biometric: </Text>
                <Badge size="xs" color={pairing.app_biometric_enabled ? "teal" : "gray"} variant="light">
                  {pairing.app_biometric_enabled ? "on" : "off"}
                </Badge>
              </Text>
              <Text size="sm">
                <Text span c="dimmed">Settings: </Text>
                <Badge size="xs" color={pairing.app_settings_locked ? "orange" : "gray"} variant="light">
                  {pairing.app_settings_locked ? "locked" : "user-editable"}
                </Badge>
              </Text>
            </Stack>
          </div>

          <Divider />

          <Button
            variant="subtle"
            color="red"
            size="sm"
            onClick={handleRevoke}
            loading={deleteMutation.isPending}
          >
            Revoke code
          </Button>
        </Stack>
      </Box>
    </Group>
  );
}
