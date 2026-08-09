import { useRef } from "react";
import { QRCodeCanvas, QRCodeSVG } from "qrcode.react";
import {
  Alert,
  Box,
  Button,
  Code,
  Divider,
  Group,
  List,
  Modal,
  Stack,
  Text,
  UnstyledButton,
} from "@mantine/core";
import { useDisclosure, useMediaQuery } from "@mantine/hooks";
import {
  IconAlertTriangle,
  IconCopy,
  IconInfoCircle,
  IconMaximize,
  IconTrash,
} from "@tabler/icons-react";
import { notifications } from "@mantine/notifications";
import type { DevicePairing } from "@/lib/api";
import { toErrorMessage } from "@/lib/api-client";
import { useClipboard } from "@/hooks/useClipboard";
import { useDeleteDevicePairing } from "./hooks/useDeleteDevicePairing";
import { PairingConfigSummary } from "./PairingConfigSummary";
import { formatTtl } from "./utils/formatTtl";

interface Props {
  deviceId: number;
  ownerId: number;
  pairing: DevicePairing;
  onRevoke: () => void;
  /** When this code replaces an already-claimed link, reassure that the old key keeps working until claim. */
  isRepair?: boolean;
}

/** Pixel size of the off-screen canvas the clipboard copy is rasterised from. */
const QR_COPY_SIZE = 512;

export function PairingCodeDisplay({ deviceId, ownerId, pairing, onRevoke, isRepair = false }: Props) {
  const { copy, copyImage } = useClipboard();
  const deleteMutation = useDeleteDevicePairing(deviceId, ownerId);
  const qrCanvasRef = useRef<HTMLCanvasElement>(null);
  const [zoomed, { open: openZoom, close: closeZoom }] = useDisclosure(false);
  const isPhone = useMediaQuery("(max-width: 36em)");

  function handleCopyQr() {
    void copyImage(
      () =>
        new Promise<Blob>((resolve, reject) => {
          const canvas = qrCanvasRef.current;
          if (!canvas) {
            reject(new Error("QR canvas not mounted"));
            return;
          }
          canvas.toBlob(
            (blob) => (blob ? resolve(blob) : reject(new Error("QR code could not be rendered"))),
            "image/png",
          );
        }),
      { successMessage: "QR code copied — paste it into a chat" },
    );
  }

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
        <Stack gap={6} align="center">
          {/* White quiet zone so the code scans on dark backgrounds */}
          <UnstyledButton
            onClick={openZoom}
            aria-label="Enlarge QR code"
            style={{ background: "white", padding: 8, borderRadius: 8, lineHeight: 0, cursor: "zoom-in" }}
          >
            <QRCodeSVG value={pairing.pairing_code} size={104} title="Pairing code QR" />
          </UnstyledButton>
          <Group gap={4}>
            <Button
              variant="subtle"
              size="xs"
              leftSection={<IconMaximize size={13} />}
              onClick={openZoom}
            >
              Enlarge
            </Button>
            <Button
              variant="subtle"
              size="xs"
              leftSection={<IconCopy size={13} />}
              onClick={handleCopyQr}
            >
              Copy QR
            </Button>
          </Group>
        </Stack>
      </Group>

      {/* Rasterisation source for "Copy QR" — a canvas the clipboard can read a
          PNG out of, kept larger than the on-screen copy so the paste stays sharp. */}
      <QRCodeCanvas
        ref={qrCanvasRef}
        value={pairing.pairing_code}
        size={QR_COPY_SIZE}
        marginSize={4}
        aria-hidden
        style={{ display: "none" }}
      />

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

      {/* Scan-from-the-screen view — full screen on a phone, where the admin panel
          itself is the thing being scanned from. */}
      <Modal
        opened={zoomed}
        onClose={closeZoom}
        title="Pairing QR code"
        size="lg"
        fullScreen={isPhone}
        centered
      >
        <Stack align="center" gap="md">
          <Box
            style={{
              background: "white",
              padding: 16,
              borderRadius: 12,
              lineHeight: 0,
              width: "100%",
              // Square, so capping the width also keeps it inside a short viewport.
              maxWidth: "min(420px, 60vh)",
            }}
          >
            <QRCodeSVG
              value={pairing.pairing_code}
              size={QR_COPY_SIZE}
              title="Pairing code QR"
              style={{ width: "100%", height: "auto" }}
            />
          </Box>
          <Code style={{ fontSize: 14, fontWeight: 600, wordBreak: "break-all" }}>
            {pairing.pairing_code}
          </Code>
          <Group gap="sm">
            <Button
              variant="default"
              size="sm"
              leftSection={<IconCopy size={14} />}
              onClick={() => copy(pairing.pairing_code, { successMessage: "Pairing code copied" })}
            >
              Copy code
            </Button>
            <Button
              variant="default"
              size="sm"
              leftSection={<IconCopy size={14} />}
              onClick={handleCopyQr}
            >
              Copy QR
            </Button>
          </Group>
        </Stack>
      </Modal>
    </Stack>
  );
}
