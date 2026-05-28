import { useState } from "react";
import {
  Alert,
  Button,
  Card,
  Divider,
  Group,
  Modal,
  Select,
  Skeleton,
  Stack,
  Text,
  TextInput,
  Title,
} from "@mantine/core";
import { notifications } from "@mantine/notifications";
import dayjs from "dayjs";
import { toErrorMessage } from "@/lib/api-client";
import { useClipboard } from "@/hooks/useClipboard";
import { useRegenerateApiKey } from "@/features/devices/hooks/useRegenerateApiKey";
import { useDeleteApiKey } from "@/features/devices/hooks/useDeleteApiKey";
import { useDeleteDevice } from "@/features/devices/hooks/useDeleteDevice";
import { useDeviceTypes } from "@/features/devices/hooks/useDeviceTypes";
import { useUpdateDevice } from "@/features/devices/hooks/useUpdateDevice";
import { useCurrentUser } from "@/features/auth/hooks/useCurrentUser";
import { useListUsers } from "@/features/auth/hooks/useListUsers";
import { DeviceProfileCard } from "@/features/devices/DeviceProfileCard";
import type { DeviceType } from "@/features/devices/deviceTypeConfig";

export interface DeviceData {
  name: string;
  api_key_prefix?: string | null;
  device_type: DeviceType;
  description?: string | null;
  icon?: string | null;
  owner_id?: number;
  owner_name?: string;
  created_at?: string | null;
}

interface DeviceSettingsTabProps {
  deviceId: number;
  device?: DeviceData;
  onDeviceDeleted?: () => void;
}

function formatCreatedAt(iso: string): string {
  return dayjs(iso).format("D MMM YYYY · HH:mm");
}

export function DeviceSettingsTab({
  deviceId,
  device,
  onDeviceDeleted,
}: DeviceSettingsTabProps) {
  const { data: deviceTypes } = useDeviceTypes();
  const regenerateApiKey = useRegenerateApiKey();
  const deleteApiKey = useDeleteApiKey();
  const deleteDevice = useDeleteDevice();
  const updateDevice = useUpdateDevice();
  const { copy } = useClipboard();

  const { data: currentUser } = useCurrentUser();
  const { data: users, isLoading: usersLoading } = useListUsers({
    enabled: currentUser != null,
  });

  const hasApiKey = Boolean(device?.api_key_prefix);

  // API key modals
  const [confirmRegenOpen, setConfirmRegenOpen] = useState(false);
  const [confirmRemoveOpen, setConfirmRemoveOpen] = useState(false);
  const [revealedApiKey, setRevealedApiKey] = useState<string | null>(null);
  const [wasRegenerated, setWasRegenerated] = useState(false);

  // Transfer ownership modal
  const [transferOpen, setTransferOpen] = useState(false);
  const [draftNewOwnerId, setDraftNewOwnerId] = useState<string | null>(null);

  // Delete device modal
  const [deleteDeviceOpen, setDeleteDeviceOpen] = useState(false);

  function handleGenerate() {
    regenerateApiKey.mutate(
      { path: { device_id: deviceId } },
      {
        onSuccess: (data) => {
          setWasRegenerated(false);
          setRevealedApiKey(data.api_key);
        },
        onError: (err) =>
          notifications.show({ color: "red", message: toErrorMessage(err) }),
      },
    );
  }

  function handleConfirmRegenerate() {
    regenerateApiKey.mutate(
      { path: { device_id: deviceId } },
      {
        onSuccess: (data) => {
          setConfirmRegenOpen(false);
          setWasRegenerated(true);
          setRevealedApiKey(data.api_key);
        },
        onError: (err) =>
          notifications.show({ color: "red", message: toErrorMessage(err) }),
      },
    );
  }

  function handleConfirmRemove() {
    deleteApiKey.mutate(
      { path: { device_id: deviceId } },
      {
        onSuccess: () => {
          setConfirmRemoveOpen(false);
          notifications.show({ color: "green", message: "API key removed" });
        },
        onError: (err) =>
          notifications.show({ color: "red", message: toErrorMessage(err) }),
      },
    );
  }

  function handleConfirmTransfer() {
    if (!draftNewOwnerId) return;
    updateDevice.mutate(
      {
        path: { device_id: deviceId },
        body: { owner_id: Number(draftNewOwnerId) },
      },
      {
        onSuccess: () => {
          setTransferOpen(false);
          setDraftNewOwnerId(null);
          notifications.show({ color: "green", message: "Device ownership transferred" });
        },
        onError: (err) => {
          const status =
            err && typeof err === "object" && "status" in err
              ? (err as { status: unknown }).status
              : undefined;
          setTransferOpen(false);
          notifications.show({
            color: "red",
            message:
              status === 403
                ? "Admin permission required to reassign ownership"
                : toErrorMessage(err),
          });
        },
      },
    );
  }

  function handleConfirmDelete() {
    deleteDevice.mutate(
      { path: { device_id: deviceId } },
      {
        onSuccess: () => {
          setDeleteDeviceOpen(false);
          notifications.show({ color: "green", message: "Device deleted" });
          onDeviceDeleted?.();
        },
        onError: (err) =>
          notifications.show({ color: "red", message: toErrorMessage(err) }),
      },
    );
  }

  const transferSelectData =
    users
      ?.filter((u) => u.id !== device?.owner_id)
      .map((u) => ({ value: String(u.id), label: u.display_name })) ?? [];

  const newOwnerName = users?.find((u) => String(u.id) === draftNewOwnerId)?.display_name;

  return (
    <Stack gap="xl">
      {/* ── 1. Profile ── */}
      <Stack gap="xs">
        <Title order={5}>Profile</Title>
        {device ? (
          <DeviceProfileCard
            deviceId={deviceId}
            device={device}
            deviceTypes={deviceTypes ?? []}
          />
        ) : (
          <Card withBorder>
            <Stack gap={8}>
              <Skeleton height={36} />
              <Skeleton height={36} />
              <Skeleton height={60} />
            </Stack>
          </Card>
        )}
        {device && (
          <Group gap="xs" px={2}>
            <Text size="xs" c="dimmed" style={{ width: 52 }}>Owner</Text>
            <Text size="xs">{device.owner_name ?? "—"}</Text>
            {device.created_at && (
              <>
                <Text size="xs" c="dimmed">·</Text>
                <Text size="xs" c="dimmed" style={{ width: 52 }}>Created</Text>
                <Text size="xs" ff="monospace">{formatCreatedAt(device.created_at)}</Text>
              </>
            )}
          </Group>
        )}
      </Stack>

      {/* ── 2. API key ── */}
      <Stack gap="xs">
        <Title order={5}>API key</Title>
        <Card withBorder>
          {!device ? (
            <Skeleton height={36} />
          ) : hasApiKey ? (
            <Stack gap={6}>
              <Group justify="space-between" wrap="nowrap">
                <Text ff="monospace" size="sm" c="dimmed">
                  {device.api_key_prefix}&hellip;
                </Text>
                <Group gap="xs" wrap="nowrap">
                  <Button
                    variant="outline"
                    size="sm"
                    disabled={regenerateApiKey.isPending}
                    onClick={() => setConfirmRegenOpen(true)}
                  >
                    ↻ Regenerate…
                  </Button>
                  <Button
                    variant="subtle"
                    color="red"
                    size="sm"
                    disabled={deleteApiKey.isPending}
                    onClick={() => setConfirmRemoveOpen(true)}
                  >
                    Remove key
                  </Button>
                </Group>
              </Group>
              <Text size="xs" c="dimmed">
                Full key shown once at creation · store it securely.
              </Text>
            </Stack>
          ) : (
            <Group justify="space-between" align="center">
              <Text size="sm" c="dimmed">
                No API key — this device cannot receive heartbeats.
              </Text>
              <Button
                size="sm"
                disabled={regenerateApiKey.isPending}
                loading={regenerateApiKey.isPending}
                onClick={handleGenerate}
              >
                Generate key
              </Button>
            </Group>
          )}
        </Card>
      </Stack>

      {/* ── 3. Danger zone ── */}
      <Stack gap="xs">
        <Title order={5} c="red.7">Danger zone</Title>
        <Card
          withBorder
          p={0}
          style={{
            borderColor: "var(--mantine-color-red-4)",
            background: "color-mix(in srgb, var(--mantine-color-red-9) 8%, transparent)",
          }}
        >
          <Group justify="space-between" align="flex-start" p="md" wrap="nowrap" gap="xl">
            <Stack gap={2}>
              <Text size="sm" fw={500}>Transfer ownership</Text>
              <Text size="xs" c="dimmed">
                Move this device to a different owner. The API key, addresses, and rules all
                transfer with it. You will lose access.
              </Text>
            </Stack>
            <Button
              variant="outline"
              size="sm"
              style={{ flexShrink: 0 }}
              onClick={() => setTransferOpen(true)}
            >
              Transfer…
            </Button>
          </Group>

          <Divider variant="dashed" color="red.3" />

          <Group justify="space-between" align="flex-start" p="md" wrap="nowrap" gap="xl">
            <Stack gap={2}>
              <Text size="sm" fw={500}>Delete device</Text>
              <Text size="xs" c="dimmed">
                All addresses become unreachable immediately. The device cannot be recovered.
                Audit history is preserved.
              </Text>
            </Stack>
            <Button
              variant="outline"
              color="red"
              size="sm"
              style={{ flexShrink: 0 }}
              onClick={() => setDeleteDeviceOpen(true)}
            >
              Delete device…
            </Button>
          </Group>
        </Card>
      </Stack>

      {/* ── Regenerate API key confirm ── */}
      <Modal
        opened={confirmRegenOpen}
        onClose={() => setConfirmRegenOpen(false)}
        title={`Regenerate API key for "${device?.name}"?`}
        closeOnClickOutside={false}
        closeOnEscape={false}
        withCloseButton={false}
      >
        <Stack gap="md">
          <Text size="sm">
            The current key (
            <Text component="span" ff="monospace">
              {device?.api_key_prefix}&hellip;
            </Text>
            ) will stop working immediately. Any device or script using it will need to be updated.
          </Text>
          <Group justify="flex-end" gap="sm">
            <Button variant="outline" onClick={() => setConfirmRegenOpen(false)}>
              Cancel
            </Button>
            <Button
              onClick={handleConfirmRegenerate}
              disabled={regenerateApiKey.isPending}
              loading={regenerateApiKey.isPending}
            >
              Regenerate
            </Button>
          </Group>
        </Stack>
      </Modal>

      {/* ── Remove API key confirm ── */}
      <Modal
        opened={confirmRemoveOpen}
        onClose={() => setConfirmRemoveOpen(false)}
        title={`Remove API key for "${device?.name}"?`}
        closeOnClickOutside={false}
        closeOnEscape={false}
        withCloseButton={false}
      >
        <Stack gap="md">
          <Text size="sm">
            The current key (
            <Text component="span" ff="monospace">
              {device?.api_key_prefix}&hellip;
            </Text>
            ) will stop working immediately and this device will no longer be able to receive
            heartbeats. You can generate a new key later.
          </Text>
          <Group justify="flex-end" gap="sm">
            <Button variant="outline" onClick={() => setConfirmRemoveOpen(false)}>
              Cancel
            </Button>
            <Button
              color="red"
              onClick={handleConfirmRemove}
              disabled={deleteApiKey.isPending}
              loading={deleteApiKey.isPending}
            >
              Delete key
            </Button>
          </Group>
        </Stack>
      </Modal>

      {/* ── One-time key reveal ── */}
      <Modal
        opened={revealedApiKey !== null}
        onClose={() => setRevealedApiKey(null)}
        title="API key generated — save it"
        closeOnClickOutside={false}
        closeOnEscape={false}
        withCloseButton={false}
      >
        <Stack gap="md">
          <Text size="sm" c="dimmed">
            This API key is shown only once. Copy it now and store it securely.
            {wasRegenerated && " The old key is no longer valid."}
          </Text>
          {revealedApiKey && (
            <>
              <Stack gap={8}>
                <Text size="sm" fw={500}>
                  {wasRegenerated ? "New API key" : "API key"}
                </Text>
                <Group gap="sm">
                  <TextInput
                    readOnly
                    value={revealedApiKey}
                    ff="monospace"
                    style={{ flex: 1 }}
                  />
                  <Button
                    type="button"
                    variant="outline"
                    onClick={() =>
                      revealedApiKey &&
                      copy(revealedApiKey, { errorMessage: "Failed to copy API key" })
                    }
                  >
                    Copy
                  </Button>
                </Group>
              </Stack>
              <Text size="xs" c="dimmed">
                You will not be able to see this full API key again. Make sure you have stored
                it securely.
              </Text>
            </>
          )}
          <Group justify="flex-end">
            <Button type="button" onClick={() => setRevealedApiKey(null)}>
              I&apos;ve saved it
            </Button>
          </Group>
        </Stack>
      </Modal>

      {/* ── Transfer ownership ── */}
      <Modal
        opened={transferOpen}
        onClose={() => {
          setTransferOpen(false);
          setDraftNewOwnerId(null);
        }}
        title={`Transfer "${device?.name}"`}
        closeOnClickOutside={!updateDevice.isPending}
      >
        <Stack gap="md">
          <Group gap="xs">
            <Text size="sm" c="dimmed" style={{ width: 36 }}>From</Text>
            <Text size="sm" fw={500}>{device?.owner_name ?? "—"}</Text>
          </Group>
          <Group gap="xs" align="center">
            <Text size="sm" c="dimmed" style={{ width: 36 }}>To</Text>
            {usersLoading ? (
              <Skeleton height={36} style={{ flex: 1 }} />
            ) : (
              <Select
                placeholder="Select new owner…"
                data={transferSelectData}
                value={draftNewOwnerId}
                onChange={setDraftNewOwnerId}
                searchable
                style={{ flex: 1 }}
              />
            )}
          </Group>
          <Alert color="yellow" variant="light" p="sm">
            <Text size="sm">
              <strong>Note:</strong> this device&apos;s live IPs will immediately be governed
              by <strong>{newOwnerName ?? "the new owner"}</strong>&apos;s host access policies.
              If their access profile differs from yours, some hosts may stop being reachable
              straight away while new ones will become reachable.
            </Text>
          </Alert>
          <Group justify="flex-end" gap="sm">
            <Button
              variant="outline"
              disabled={updateDevice.isPending}
              onClick={() => {
                setTransferOpen(false);
                setDraftNewOwnerId(null);
              }}
            >
              Cancel
            </Button>
            <Button
              disabled={!draftNewOwnerId || updateDevice.isPending}
              loading={updateDevice.isPending}
              onClick={handleConfirmTransfer}
            >
              Transfer device
            </Button>
          </Group>
        </Stack>
      </Modal>

      {/* ── Delete device ── */}
      <Modal
        opened={deleteDeviceOpen}
        onClose={() => setDeleteDeviceOpen(false)}
        title={`Delete "${device?.name}"?`}
        closeOnClickOutside={false}
        closeOnEscape={false}
        withCloseButton={false}
      >
        <Stack gap="md">
          <Text size="sm">
            All addresses for this device will stop responding immediately. The device cannot
            be recovered. Audit history is preserved.
          </Text>
          <Group justify="flex-end" gap="sm">
            <Button
              variant="outline"
              disabled={deleteDevice.isPending}
              onClick={() => setDeleteDeviceOpen(false)}
            >
              Cancel
            </Button>
            <Button
              color="red"
              onClick={handleConfirmDelete}
              disabled={deleteDevice.isPending}
              loading={deleteDevice.isPending}
            >
              Delete device
            </Button>
          </Group>
        </Stack>
      </Modal>
    </Stack>
  );
}
