import dayjs from "dayjs";
import { QRCodeSVG } from "qrcode.react";
import {
  Alert,
  Box,
  Button,
  Code,
  Divider,
  Group,
  List,
  Stack,
  Text,
} from "@mantine/core";
import { IconAlertTriangle, IconCopy, IconInfoCircle, IconTrash } from "@tabler/icons-react";
import { notifications } from "@mantine/notifications";
import type { DevicePairing } from "@/lib/api";
import { toErrorMessage } from "@/lib/api-client";
import { useClipboard } from "@/hooks/useClipboard";
import { useDeleteDevicePairing } from "./hooks/useDeleteDevicePairing";
import { PairingConfigSummary } from "./PairingConfigSummary";

function formatTtl(expiresAt: string): string {
  const diffMin = dayjs(expiresAt).diff(dayjs(), "minute");
  if (diffMin <= 0) return "expired";
  if (diffMin < 60) return `${diffMin}m remaining`;
  const h = Math.floor(diffMin / 60);
  const m = diffMin % 60;
  return m > 0 ? `${h}h ${m}m remaining` : `${h}h remaining`;
}

interface Props {
  deviceId: number;
  pairing: DevicePairing;
  onRevoke: () => void;
  /** When this code replaces an already-claimed link, reassure that the old key keeps working until claim. */
  isRepair?: boolean;
}

export function PairingCodeDisplay({ deviceId, pairing, onRevoke, isRepair = false }: Props) {
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
      {isRepair && (
        <Alert color="indigo" icon={<IconInfoCircle size={16} />}>
          The current link stays active until this new code is claimed — the device keeps working
          on its existing key in the meantime.
        </Alert>
      )}

      {/* 1. The code — primary focus */}
      <Group gap="lg" align="flex-start" wrap="wrap">
        <div style={{ flex: 1, minWidth: 220 }}>
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
        <Stack gap={4} align="center">
          {/* White quiet zone so the code scans on dark backgrounds */}
          <Box
            aria-label="QR code with the pairing code"
            style={{ background: "white", padding: 8, borderRadius: 8, lineHeight: 0 }}
          >
            <QRCodeSVG value={pairing.pairing_code} size={104} />
          </Box>
          <Text size="xs" c="dimmed">
            Scan to copy on a phone
          </Text>
        </Stack>
      </Group>

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
      <PairingConfigSummary pairing={pairing} />

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
