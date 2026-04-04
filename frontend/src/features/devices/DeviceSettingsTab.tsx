import { useState } from "react";
import {
  Button,
  Card,
  Group,
  Modal,
  Skeleton,
  Stack,
  Text,
  TextInput,
  Title,
} from "@mantine/core";
import { notifications } from "@mantine/notifications";
import { toErrorMessage } from "@/lib/api-client";
import { useRegenerateApiKey } from "@/features/devices/hooks/useRegenerateApiKey";
import { useDeviceTypes } from "@/features/devices/hooks/useDeviceTypes";
import { DeviceProfileCard } from "@/features/devices/DeviceProfileCard";
import { DeviceOwnershipCard } from "@/features/devices/DeviceOwnershipCard";
import { AddressLeaseRuleCard } from "@/features/devices/AddressLeaseRuleCard";
import { MaxActiveIpsRuleCard } from "@/features/devices/MaxActiveIpsRuleCard";
import type { DeviceType } from "@/features/devices/deviceTypeConfig";

export interface DeviceData {
  name: string;
  api_key_prefix: string;
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

export function DeviceSettingsTab({
  deviceId,
  device,
}: DeviceSettingsTabProps) {
  const { data: deviceTypes } = useDeviceTypes();
  const regenerateApiKey = useRegenerateApiKey();

  const [regeneratedApiKey, setRegeneratedApiKey] = useState<string | null>(
    null,
  );
  const [confirmRegenOpen, setConfirmRegenOpen] = useState(false);

  async function handleCopyRegeneratedKey() {
    if (!regeneratedApiKey) return;
    if (!("clipboard" in navigator) || !navigator.clipboard?.writeText) {
      notifications.show({
        message: "Copy to clipboard is not supported in this browser.",
        color: "red",
      });
      return;
    }
    try {
      await navigator.clipboard.writeText(regeneratedApiKey);
      notifications.show({ message: "Copied to clipboard", color: "green" });
    } catch {
      notifications.show({ message: "Failed to copy API key", color: "red" });
    }
  }

  function handleConfirmRegenerate() {
    regenerateApiKey.mutate(
      { path: { device_id: deviceId } },
      {
        onSuccess: (data) => {
          setConfirmRegenOpen(false);
          setRegeneratedApiKey(data.api_key);
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
                <Text ff="monospace" size="sm" c="dimmed">
                  {device.api_key_prefix}&hellip;
                </Text>
              ) : (
                <Skeleton height={16} width={128} />
              )}
            </Stack>
            <Button
              variant="outline"
              size="sm"
              disabled={!device || regenerateApiKey.isPending}
              onClick={() => setConfirmRegenOpen(true)}
            >
              Regenerate API key
            </Button>
          </Group>
        </Card>
      </Stack>

      {/* Rules */}
      <Stack gap="sm">
        <Title order={5}>Rules</Title>
        <AddressLeaseRuleCard deviceId={deviceId} />
        <MaxActiveIpsRuleCard deviceId={deviceId} />
      </Stack>

      {/* Confirm regenerate API key modal */}
      <Modal
        opened={confirmRegenOpen}
        onClose={() => setConfirmRegenOpen(false)}
        title={`Regenerate API key for "${device?.name}"?`}
        closeOnClickOutside={false}
        closeOnEscape={false}
        withCloseButton={false}
      >
        <Text size="sm">
          The current key (
          <Text component="span" ff="monospace">
            {device?.api_key_prefix}&hellip;
          </Text>
          ) will stop working immediately. You will need to update any scripts
          or services using this device.
        </Text>
        <Group justify="flex-end" mt="md" gap="sm">
          <Button variant="outline" onClick={() => setConfirmRegenOpen(false)}>
            Cancel
          </Button>
          <Button
            onClick={handleConfirmRegenerate}
            disabled={regenerateApiKey.isPending}
          >
            Regenerate
          </Button>
        </Group>
      </Modal>

      {/* One-time key display modal after successful regeneration */}
      <Modal
        opened={regeneratedApiKey !== null}
        onClose={() => setRegeneratedApiKey(null)}
        title="API key regenerated — save your new key"
        closeOnClickOutside={false}
        closeOnEscape={false}
        withCloseButton={false}
      >
        <Stack gap="md">
          <Text size="sm" c="dimmed">
            This API key is shown only once. Copy it now and store it securely.
            The old key is no longer valid.
          </Text>
          {regeneratedApiKey && (
            <>
              <Stack gap={8}>
                <Text size="sm" fw={500}>
                  New API key
                </Text>
                <Group gap="sm">
                  <TextInput
                    readOnly
                    value={regeneratedApiKey}
                    ff="monospace"
                    style={{ flex: 1 }}
                  />
                  <Button
                    type="button"
                    variant="outline"
                    onClick={handleCopyRegeneratedKey}
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
            <Button type="button" onClick={() => setRegeneratedApiKey(null)}>
              I&apos;ve saved it
            </Button>
          </Group>
        </Stack>
      </Modal>
    </Stack>
  );
}
