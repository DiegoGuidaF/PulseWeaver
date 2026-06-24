import { useMemo, useState } from "react";
import {
  Alert,
  Badge,
  Button,
  Collapse,
  Divider,
  Group,
  Loader,
  Modal,
  Stack,
  Text,
  Title,
  UnstyledButton,
} from "@mantine/core";
import { useDisclosure } from "@mantine/hooks";
import { notifications } from "@mantine/notifications";
import {
  IconAlertCircle,
  IconChevronRight,
  IconClock,
  IconPlus,
  IconRefresh,
} from "@tabler/icons-react";
import dayjs from "dayjs";
import relativeTime from "dayjs/plugin/relativeTime";
import type { DevicePairing, DeviceState as DeviceStateType } from "@/lib/api";
import { DeviceState } from "@/lib/api";
import { toErrorMessage } from "@/lib/api-client";
import { ErrorState } from "@/components/ErrorState";
import { useDateFormatter } from "@/contexts/useDateTimePrefs";
import { useListDevicePairings } from "./hooks/useListDevicePairings";
import { useCreateDevicePairing } from "./hooks/useCreateDevicePairing";
import { PairingCreationForm } from "./PairingCreationForm";
import { PairingCodeDisplay } from "./PairingCodeDisplay";
import { PairingConfigSummary } from "./PairingConfigSummary";
import { PairingStatusHero } from "./PairingStatusHero";

dayjs.extend(relativeTime);

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
  const isExpired = deviceState === DeviceState.EXPIRED_CLAIM;

  const pendingQuery = useListDevicePairings(deviceId, "pending");
  const historyQuery = useListDevicePairings(deviceId, "all");
  const regenMutation = useCreateDevicePairing(deviceId);

  const formatDateTime = useDateFormatter();

  // After creation, switch immediately to the code display (the pending query
  // refreshes via invalidation behind it).
  const [justCreated, setJustCreated] = useState<DevicePairing | null>(null);
  const [createOpen, { open: openCreate, close: closeCreate }] = useDisclosure(false);
  const [showHistory, { toggle: toggleHistory }] = useDisclosure(false);

  const pendingPairing = pendingQuery.data?.[0];
  const outstandingCode = justCreated ?? pendingPairing;

  // A claimed code means the device is (or was) linked to a companion app. The
  // most recent one carries the config locked in at claim time.
  const claimedPairing = useMemo(
    () => (historyQuery.data ?? []).find((p) => p.status === "used"),
    [historyQuery.data],
  );
  const isLinked = Boolean(claimedPairing);

  const historyItems = (historyQuery.data ?? []).filter((p) => p.status !== "pending").slice(0, 5);

  function handleCreateSuccess(pairing: DevicePairing) {
    setJustCreated(pairing);
    closeCreate();
  }

  function handleRevoke() {
    setJustCreated(null);
  }

  function handleRegen() {
    if (!outstandingCode) return;
    regenMutation.mutate(
      {
        path: { id: deviceId },
        body: {
          heartbeat_server_url: outstandingCode.heartbeat_server_url,
          interval_seconds: outstandingCode.interval_seconds,
          app_biometric_enabled: outstandingCode.app_biometric_enabled,
          app_settings_locked: outstandingCode.app_settings_locked,
          expires_in_hours: 24,
        },
      },
      {
        onSuccess: (data) => {
          setJustCreated(data);
          notifications.show({ color: "green", message: "New pairing code generated" });
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

  if ((pendingQuery.isLoading || historyQuery.isLoading) && !justCreated) {
    return <Loader size="sm" />;
  }

  if ((pendingQuery.isError || historyQuery.isError) && !justCreated) {
    const failed = pendingQuery.isError ? pendingQuery : historyQuery;
    return (
      <ErrorState
        error={failed.error}
        title="Failed to load pairing status"
        onRetry={() => failed.refetch()}
      />
    );
  }

  return (
    <Stack gap="lg">
      {outstandingCode ? (
        /* A code is outstanding — for a fresh device this is first pairing, for a
           linked one it's a replacement waiting to be claimed. */
        <Stack gap="md">
          <Group justify="space-between" align="center">
            <Title order={2} size="h5">
              {isLinked ? "New pairing code" : "Active pairing code"}
            </Title>
            <Button
              variant="light"
              size="xs"
              color="orange"
              leftSection={<IconRefresh size={13} />}
              onClick={handleRegen}
              loading={regenMutation.isPending}
            >
              Regenerate
            </Button>
          </Group>
          <PairingCodeDisplay
            deviceId={deviceId}
            pairing={outstandingCode}
            onRevoke={handleRevoke}
            isRepair={isLinked}
          />
        </Stack>
      ) : isLinked && claimedPairing ? (
        /* Linked: lead with status, demote the act of issuing another code. */
        <Stack gap="md">
          <PairingStatusHero deviceState={deviceState} claimedAt={claimedPairing.updated_at} />
          <Divider />
          <Stack gap={6}>
            <Text size="sm" fw={500}>
              Config locked in at claim
            </Text>
            <PairingConfigSummary pairing={claimedPairing} />
          </Stack>
          <Group>
            <Button
              variant="default"
              size="sm"
              leftSection={<IconRefresh size={14} />}
              onClick={openCreate}
            >
              Generate another code
            </Button>
          </Group>
          <Text size="xs" c="dimmed">
            The current link stays active until a new code is claimed.
          </Text>
        </Stack>
      ) : isExpired ? (
        /* Expired without ever being claimed — one clear way forward. */
        <Stack gap="md">
          <Alert color="orange" icon={<IconAlertCircle size={16} />} title="Pairing code expired">
            The previous code expired before it was claimed. Generate a new one to pair this device.
          </Alert>
          <Group>
            <Button leftSection={<IconPlus size={14} />} onClick={openCreate}>
              Generate new code
            </Button>
          </Group>
        </Stack>
      ) : (
        /* Never linked — the create form is the main thing to do here. */
        <Stack gap="md">
          <div>
            <Title order={2} size="h5" mb={4}>
              Generate a pairing code
            </Title>
            <Text size="sm" c="dimmed">
              Create a one-time code to link the companion app to this device. Once claimed, the
              app sends heartbeats and PulseWeaver tracks the device's IP automatically.
            </Text>
          </div>
          <PairingCreationForm deviceId={deviceId} onSuccess={handleCreateSuccess} />
        </Stack>
      )}

      {/* History — collapsed by default; secondary to current status */}
      {historyItems.length > 0 && (
        <>
          <Divider />
          <Stack gap="xs">
            <UnstyledButton onClick={toggleHistory}>
              <Group gap={6} align="center">
                <IconChevronRight
                  size={14}
                  style={{
                    color: "var(--mantine-color-dimmed)",
                    transform: showHistory ? "rotate(90deg)" : "none",
                    transition: "transform 150ms ease",
                  }}
                />
                <Text size="sm" fw={600} c="dimmed">
                  Recent codes · {historyItems.length}
                </Text>
              </Group>
            </UnstyledButton>
            <Collapse expanded={showHistory}>
              <Stack gap="xs">
                {historyItems.map((item) => {
                  const badge = STATUS_BADGE[item.status];
                  return (
                    <Group key={item.id} gap="sm" wrap="nowrap">
                      <Text
                        size="xs"
                        ff="monospace"
                        c="dimmed"
                        truncate
                        style={{ flex: 1, minWidth: 0 }}
                      >
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
                            : item.status === "expired" ||
                                item.status === "invalidated" ||
                                item.status === "replaced"
                              ? formatDateTime(item.updated_at)
                              : formatDateTime(item.created_at)}
                        </Text>
                      </Group>
                    </Group>
                  );
                })}
              </Stack>
            </Collapse>
          </Stack>
        </>
      )}

      {/* Generate-code modal — used by the linked and expired states */}
      <Modal
        opened={createOpen}
        onClose={closeCreate}
        title="Generate a pairing code"
        size="lg"
        closeOnClickOutside={false}
      >
        <PairingCreationForm
          deviceId={deviceId}
          onSuccess={handleCreateSuccess}
          onCancel={closeCreate}
        />
      </Modal>
    </Stack>
  );
}
