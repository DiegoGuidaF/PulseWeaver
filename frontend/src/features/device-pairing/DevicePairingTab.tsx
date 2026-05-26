import { useState } from "react";
import {
  Alert,
  Badge,
  Button,
  Divider,
  Group,
  Loader,
  Modal,
  Stack,
  Text,
  Title,
} from "@mantine/core";
import { useDisclosure } from "@mantine/hooks";
import { notifications } from "@mantine/notifications";
import { IconAlertCircle, IconClock } from "@tabler/icons-react";
import dayjs from "dayjs";
import type { DevicePairing, DeviceState as DeviceStateType } from "@/lib/api";
import { DeviceState } from "@/lib/api";
import { toErrorMessage } from "@/lib/api-client";
import { useDateFormatter } from "@/contexts/useDateTimePrefs";
import { useListDevicePairings } from "./hooks/useListDevicePairings";
import { useCreateDevicePairing } from "./hooks/useCreateDevicePairing";
import { PairingCreationForm } from "./PairingCreationForm";
import { PairingCodeDisplay } from "./PairingCodeDisplay";

const STATUS_BADGE: Record<DevicePairing["status"], { label: string; color: string }> = {
  pending: { label: "pending", color: "indigo" },
  used: { label: "claimed", color: "green" },
  expired: { label: "expired", color: "red" },
  invalidated: { label: "revoked", color: "gray" },
  replaced: { label: "replaced", color: "gray" },
};

interface Props {
  deviceId: number;
  deviceState: DeviceStateType;
}

export function DevicePairingTab({ deviceId, deviceState }: Props) {
  const [createOpen, { open: openCreate, close: closeCreate }] = useDisclosure(false);
  const [regenOpen, { open: openRegen, close: closeRegen }] = useDisclosure(false);

  const isPending = deviceState === DeviceState.PENDING_CLAIM;
  const isExpired = deviceState === DeviceState.EXPIRED_CLAIM;

  const pendingQuery = useListDevicePairings(deviceId, "pending");
  const historyQuery = useListDevicePairings(deviceId, "all");
  const regenMutation = useCreateDevicePairing(deviceId);

  const pendingPairing = pendingQuery.data?.[0];
  const historyItems = (historyQuery.data ?? []).filter((p) => p.status !== "pending").slice(0, 5);

  const formatDateTime = useDateFormatter();

  // After creation, switch immediately to the display (pending query will refresh via invalidation)
  const [justCreated, setJustCreated] = useState<DevicePairing | null>(null);

  const displayPairing = justCreated ?? pendingPairing;

  function handleCreateSuccess(pairing: DevicePairing) {
    setJustCreated(pairing);
    closeCreate();
  }

  function handleRevoke() {
    setJustCreated(null);
  }

  function handleRegen() {
    if (!displayPairing) return;
    regenMutation.mutate(
      {
        path: { id: deviceId },
        body: {
          heartbeat_server_url: displayPairing.heartbeat_server_url,
          interval_seconds: displayPairing.interval_seconds,
          app_biometric_enabled: displayPairing.app_biometric_enabled,
          app_settings_locked: displayPairing.app_settings_locked,
          expires_in_hours: 24,
        },
      },
      {
        onSuccess: (data) => {
          setJustCreated(data);
          closeRegen();
        },
        onError: (err) =>
          notifications.show({
            color: "red",
            title: "Failed to generate new pairing code",
            message: toErrorMessage(err),
          }),
      },
    );
  }

  if (isPending && pendingQuery.isLoading && !justCreated) {
    return <Loader size="sm" />;
  }

  return (
    <Stack gap="lg">
      {/* Active code display (pending state) */}
      {displayPairing ? (
        <Stack gap="md">
          <Group justify="space-between" align="center">
            <Title order={5}>Active pairing code</Title>
            <Button variant="subtle" size="xs" color="orange" onClick={openRegen}>
              Regenerate
            </Button>
          </Group>
          <PairingCodeDisplay
            deviceId={deviceId}
            pairing={displayPairing}
            onRevoke={handleRevoke}
          />
        </Stack>
      ) : (
        <Stack gap="sm">
          {isExpired && (
            <Alert color="orange" icon={<IconAlertCircle size={16} />}>
              The previous pairing code expired before it was claimed. Generate a new one below.
            </Alert>
          )}
          <Group>
            <Button onClick={openCreate}>Generate pairing code</Button>
          </Group>
        </Stack>
      )}

      {/* History */}
      {historyItems.length > 0 && (
        <>
          <Divider />
          <Stack gap="xs">
            <Title order={6} c="dimmed">
              Recent codes
            </Title>
            {historyItems.map((item) => {
              const badge = STATUS_BADGE[item.status];
              return (
                <Group key={item.id} gap="sm" wrap="nowrap">
                  <Text size="xs" ff="monospace" style={{ flex: 1, minWidth: 0 }}>
                    {item.pairing_code}
                  </Text>
                  <Badge size="xs" color={badge.color} variant="light" style={{ flexShrink: 0 }}>
                    {badge.label}
                  </Badge>
                  <Group gap={4} style={{ flexShrink: 0 }}>
                    <IconClock size={11} style={{ color: "var(--mantine-color-dimmed)" }} />
                    <Text size="xs" c="dimmed">
                      {item.status === "used"
                        ? `claimed ${dayjs(item.updated_at).fromNow()}`
                        : item.status === "expired" || item.status === "invalidated" || item.status === "replaced"
                          ? formatDateTime(item.updated_at)
                          : formatDateTime(item.created_at)}
                    </Text>
                  </Group>
                </Group>
              );
            })}
          </Stack>
        </>
      )}

      {/* Create modal */}
      <Modal
        opened={createOpen}
        onClose={closeCreate}
        title="Generate pairing code"
        size="md"
        closeOnClickOutside={false}
      >
        <PairingCreationForm
          deviceId={deviceId}
          onSuccess={handleCreateSuccess}
          onCancel={closeCreate}
        />
      </Modal>

      {/* Regenerate confirm modal */}
      <Modal
        opened={regenOpen}
        onClose={closeRegen}
        title="Regenerate pairing code?"
        size="sm"
      >
        <Stack gap="md">
          <Text size="sm">
            Generating a new code will invalidate the current one. If nobody has
            claimed it yet, the device's API key is unchanged.
          </Text>
          <Group justify="flex-end" gap="sm">
            <Button variant="outline" onClick={closeRegen}>
              Cancel
            </Button>
            <Button
              color="orange"
              loading={regenMutation.isPending}
              onClick={handleRegen}
            >
              Generate new code
            </Button>
          </Group>
        </Stack>
      </Modal>
    </Stack>
  );
}
