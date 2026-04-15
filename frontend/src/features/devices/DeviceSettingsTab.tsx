import {useState} from "react";
import {ActionIcon, Button, Card, Group, Modal, Skeleton, Stack, Text, TextInput, Title, Tooltip,} from "@mantine/core";
import {IconTrash} from "@tabler/icons-react";
import {notifications} from "@mantine/notifications";
import {toErrorMessage} from "@/lib/api-client";
import {useClipboard} from "@/hooks/useClipboard";
import {useRegenerateApiKey} from "@/features/devices/hooks/useRegenerateApiKey";
import {useDeleteApiKey} from "@/features/devices/hooks/useDeleteApiKey";
import {useDeviceTypes} from "@/features/devices/hooks/useDeviceTypes";
import {DeviceProfileCard} from "@/features/devices/DeviceProfileCard";
import {DeviceOwnershipCard} from "@/features/devices/DeviceOwnershipCard";
import type {DeviceType} from "@/features/devices/deviceTypeConfig";

export interface DeviceData {
  name: string;
  api_key_prefix?: string | null;
  device_type: DeviceType;
  description?: string | null;
  icon?: string | null;
  owner_id?: number;
  owner_name?: string;
}

interface DeviceSettingsTabProps {
  deviceId: number;
  device?: DeviceData;
}

const HEARTBEAT_WARNING =
  "Any currently connected device using this api key will immediately stop being able to send heartbeats via the API.";

export function DeviceSettingsTab({
  deviceId,
  device,
}: DeviceSettingsTabProps) {
  const { data: deviceTypes } = useDeviceTypes();
  const regenerateApiKey = useRegenerateApiKey();
  const deleteApiKey = useDeleteApiKey();
  const { copy } = useClipboard();

  const hasApiKey = Boolean(device?.api_key_prefix);

  // Modal visibility
  const [confirmRegenOpen, setConfirmRegenOpen] = useState(false);
  const [confirmDeleteOpen, setConfirmDeleteOpen] = useState(false);

  // One-time reveal after generate/regenerate
  const [revealedApiKey, setRevealedApiKey] = useState<string | null>(null);
  const [wasRegenerated, setWasRegenerated] = useState(false);

  function handleGenerate() {
    // No confirm modal for first-time generation — nothing to invalidate.
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

  function handleConfirmDelete() {
    deleteApiKey.mutate(
      { path: { device_id: deviceId } },
      {
        onSuccess: () => {
          setConfirmDeleteOpen(false);
          notifications.show({
            color: "green",
            message: "API key removed. The device can no longer receive heartbeats via API.",
          });
        },
        onError: (err) =>
          notifications.show({ color: "red", message: toErrorMessage(err) }),
      },
    );
  }

  return (
    <Stack gap="xl">
      {/* Device profile */}
      <Stack gap="sm">
        <Title order={5}>Device profile</Title>
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
      </Stack>

      {/* Ownership */}
      <Stack gap="sm">
        <Title order={5}>Ownership</Title>
        {device ? (
          <DeviceOwnershipCard
            deviceId={deviceId}
            ownerId={device.owner_id}
            ownerName={device.owner_name}
          />
        ) : (
          <Card withBorder>
            <Skeleton height={20} width={180} />
          </Card>
        )}
      </Stack>

      {/* Settings */}
      <Stack gap="sm">
        <Title order={5}>Settings</Title>
        <Card withBorder>
          <Group justify="space-between" gap="md">
            <Stack gap={4}>
              <Text size="sm" fw={500}>
                API Key
              </Text>
              {device ? (
                hasApiKey ? (
                  <Text ff="monospace" size="sm" c="dimmed">
                    {device.api_key_prefix}&hellip;
                  </Text>
                ) : (
                  <Text size="sm" c="dimmed">
                    No API key
                  </Text>
                )
              ) : (
                <Skeleton height={16} width={128} />
              )}
            </Stack>

            {hasApiKey ? (
              <Group gap="xs">
                <Button
                  variant="outline"
                  size="sm"
                  disabled={!device || regenerateApiKey.isPending}
                  onClick={() => setConfirmRegenOpen(true)}
                >
                  Regenerate API key
                </Button>
                <Tooltip label="Remove API key" withArrow>
                  <ActionIcon
                    variant="subtle"
                    color="red"
                    size="md"
                    aria-label="Remove API key"
                    disabled={!device || deleteApiKey.isPending}
                    onClick={() => setConfirmDeleteOpen(true)}
                  >
                    <IconTrash size={16} />
                  </ActionIcon>
                </Tooltip>
              </Group>
            ) : (
              <Button
                size="sm"
                disabled={!device || regenerateApiKey.isPending}
                loading={regenerateApiKey.isPending}
                onClick={handleGenerate}
              >
                Generate API key
              </Button>
            )}
          </Group>
        </Card>
      </Stack>

      {/* Regenerate confirm modal */}
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
            The current api key (
            <Text component="span" ff="monospace">
              {device?.api_key_prefix}&hellip;
            </Text>
            ) will stop working immediately. {HEARTBEAT_WARNING}
          </Text>
          <Text size="sm">
            You will need to update any scripts or services using this device.
          </Text>
          <Group justify="flex-end" gap="sm">
            <Button
              variant="outline"
              onClick={() => setConfirmRegenOpen(false)}
            >
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

      {/* Delete key confirm modal */}
      <Modal
        opened={confirmDeleteOpen}
        onClose={() => setConfirmDeleteOpen(false)}
        title={`Remove API key for "${device?.name}"?`}
        closeOnClickOutside={false}
        closeOnEscape={false}
        withCloseButton={false}
      >
        <Stack gap="md">
          <Text size="sm">
            The current api key (
            <Text component="span" ff="monospace">
              {device?.api_key_prefix}&hellip;
            </Text>
            ) will be permanently revoked. {HEARTBEAT_WARNING}
          </Text>
          <Text size="sm">
            A new key can be generated at any time from this page.
          </Text>
          <Group justify="flex-end" gap="sm">
            <Button
              variant="outline"
              onClick={() => setConfirmDeleteOpen(false)}
            >
              Cancel
            </Button>
            <Button
              color="red"
              onClick={handleConfirmDelete}
              disabled={deleteApiKey.isPending}
              loading={deleteApiKey.isPending}
            >
              Delete key
            </Button>
          </Group>
        </Stack>
      </Modal>

      {/* One-time key reveal modal after generate / regenerate */}
      <Modal
        opened={revealedApiKey !== null}
        onClose={() => setRevealedApiKey(null)}
        title={ "API key generated — save it" }
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
                      copy(revealedApiKey, {
                        errorMessage: "Failed to copy API key",
                      })
                    }
                  >
                    Copy
                  </Button>
                </Group>
              </Stack>
              <Text size="xs" c="dimmed">
                You will not be able to see this full API key again. Make sure
                you have stored it securely.
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
    </Stack>
  );
}
