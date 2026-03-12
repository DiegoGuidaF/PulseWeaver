import { useEffect, useState } from "react";
import { useForm } from "@mantine/form";
import { zod4Resolver } from "mantine-form-zod-resolver";
import { z } from "zod";
import {
  Button,
  Card,
  Group,
  Modal,
  NativeSelect,
  Skeleton,
  Stack,
  Text,
  TextInput,
  Title,
} from "@mantine/core";
import { notifications } from "@mantine/notifications";
import { toErrorMessage } from "@/lib/api-client";
import { useDeviceAddressLeaseRule } from "@/features/devices/hooks/useDeviceAddressLeaseRule";
import { usePutDeviceAddressLeaseRule } from "@/features/devices/hooks/usePutDeviceAddressLeaseRule";
import { useDisableDeviceAddressLeaseRule } from "@/features/devices/hooks/useDisableDeviceAddressLeaseRule";
import { useRegenerateApiKey } from "@/features/devices/hooks/useRegenerateApiKey";

const TTL_UNITS = ["seconds", "minutes", "hours", "days"] as const;
const SECONDS_PER_MINUTE = 60;
const SECONDS_PER_HOUR = 3600;
const SECONDS_PER_DAY = 86400;

type TtlUnit = (typeof TTL_UNITS)[number];

function toSeconds(value: number, unit: TtlUnit): number {
  switch (unit) {
    case "seconds":
      return value;
    case "minutes":
      return value * SECONDS_PER_MINUTE;
    case "hours":
      return value * SECONDS_PER_HOUR;
    case "days":
      return value * SECONDS_PER_DAY;
    default: {
      const _exhaustive: never = unit;
      return _exhaustive;
    }
  }
}

function fromSeconds(ttlSeconds: number): { value: string; unit: TtlUnit } {
  if (ttlSeconds % SECONDS_PER_DAY === 0) {
    return { value: String(ttlSeconds / SECONDS_PER_DAY), unit: "days" };
  }
  if (ttlSeconds % SECONDS_PER_HOUR === 0) {
    return { value: String(ttlSeconds / SECONDS_PER_HOUR), unit: "hours" };
  }
  if (ttlSeconds % SECONDS_PER_MINUTE === 0) {
    return { value: String(ttlSeconds / SECONDS_PER_MINUTE), unit: "minutes" };
  }
  return { value: String(ttlSeconds), unit: "seconds" };
}

function formatTtlLabel(ttlSeconds: number): string {
  if (ttlSeconds % SECONDS_PER_DAY === 0) {
    const days = ttlSeconds / SECONDS_PER_DAY;
    return days === 1 ? "1 day" : `${days} days`;
  }
  if (ttlSeconds % SECONDS_PER_MINUTE === 0) {
    const minutes = ttlSeconds / SECONDS_PER_MINUTE;
    if (minutes % 60 === 0) {
      const hours = minutes / 60;
      return hours === 1 ? "1 hour" : `${hours} hours`;
    }
    return minutes === 1 ? "1 minute" : `${minutes} minutes`;
  }
  return ttlSeconds === 1 ? "1 second" : `${ttlSeconds} seconds`;
}

type LeaseRuleFormValues = { value: string; unit: TtlUnit };

const leaseRuleFormSchema = z.object({
  value: z.coerce.number().int("Must be a whole number").min(1, "Minimum is 1"),
  unit: z.enum(TTL_UNITS),
});

interface DeviceSettingsTabProps {
  deviceId: number;
  device?: { name: string; api_key_prefix: string };
}

export function DeviceSettingsTab({
  deviceId,
  device,
}: DeviceSettingsTabProps) {
  const {
    data: rule,
    isLoading,
    isError,
    error,
  } = useDeviceAddressLeaseRule(deviceId);
  const putRuleMutation = usePutDeviceAddressLeaseRule(deviceId);
  const disableRuleMutation = useDisableDeviceAddressLeaseRule(deviceId);
  const regenerateApiKey = useRegenerateApiKey();

  const [regeneratedApiKey, setRegeneratedApiKey] = useState<string | null>(
    null,
  );
  const [confirmRegenOpen, setConfirmRegenOpen] = useState(false);

  const leaseRuleForm = useForm<LeaseRuleFormValues>({
    validate: zod4Resolver(leaseRuleFormSchema),
    initialValues: { value: "5", unit: "minutes" },
  });
  const { setValues } = leaseRuleForm;
  const [editing, setEditing] = useState(false);

  const isOn = Boolean(rule && rule.enabled);

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

  function handleLeaseRuleSubmit(values: LeaseRuleFormValues) {
    putRuleMutation.mutate(
      {
        path: { device_id: deviceId },
        body: { ttl_seconds: toSeconds(Number(values.value), values.unit) },
      },
      {
        onSuccess: () =>
          notifications.show({
            color: "green",
            message: "Address lease rule saved",
          }),
        onError: (err) =>
          notifications.show({
            color: "red",
            title: "Error",
            message: toErrorMessage(err),
          }),
      },
    );
    setEditing(false);
  }

  function handleStartEditing() {
    if (!rule) return;
    setValues(fromSeconds(rule.ttl_seconds));
    setEditing(true);
  }

  useEffect(() => {
    if (!rule || isOn) return;
    setValues(fromSeconds(rule.ttl_seconds));
  }, [isOn, rule, setValues]);

  const ttlLabel =
    rule && rule.ttl_seconds ? formatTtlLabel(rule.ttl_seconds) : null;
  const submitButtonLabel = putRuleMutation.isPending
    ? "Saving..."
    : isOn
      ? "Save"
      : "Enable auto-expiry";

  return (
    <Stack gap="xl">
      {/* Settings section */}
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

      {/* Rules section */}
      <Stack gap="sm">
        <Title order={5}>Rules</Title>
        <Card withBorder>
          <Title order={4} mb="md">
            Auto-expiry rule
          </Title>
          {isLoading ? (
            <Stack gap={8}>
              <Skeleton height={16} width={160} />
              <Skeleton height={16} width={256} />
            </Stack>
          ) : isError ? (
            <Text size="sm" c="red">
              Error loading rule: {toErrorMessage(error)}
            </Text>
          ) : (
            <Stack gap="md">
              {isOn && (
                <Stack gap={4}>
                  <Text size="sm">
                    Status:{" "}
                    <Text component="span" fw={500}>
                      Enabled
                    </Text>
                  </Text>
                  {ttlLabel && (
                    <Text size="sm" c="dimmed">
                      Addresses will automatically expire after{" "}
                      <Text component="span" fw={500}>
                        {ttlLabel}
                      </Text>
                      .
                    </Text>
                  )}
                </Stack>
              )}

              {!isOn && (
                <Text size="sm" c="dimmed">
                  Auto-expiry is currently{" "}
                  <Text component="span" fw={500} c="var(--mantine-color-text)">
                    disabled
                  </Text>
                  . Turn it on to automatically revoke stale addresses.
                </Text>
              )}

              {(!isOn || editing) && (
                <form onSubmit={leaseRuleForm.onSubmit(handleLeaseRuleSubmit)}>
                  <Group align="flex-end" gap="md" wrap="wrap">
                    <TextInput
                      label="Expires after"
                      type="number"
                      min={1}
                      step={1}
                      placeholder="1"
                      w={128}
                      {...leaseRuleForm.getInputProps("value")}
                    />
                    <NativeSelect
                      label="Unit"
                      w={128}
                      data={TTL_UNITS.map((unit) => ({
                        label: unit,
                        value: unit,
                      }))}
                      {...leaseRuleForm.getInputProps("unit")}
                    />
                    <Button type="submit" disabled={putRuleMutation.isPending}>
                      {submitButtonLabel}
                    </Button>
                    {editing && (
                      <Button
                        type="button"
                        variant="outline"
                        onClick={() => setEditing(false)}
                      >
                        Cancel
                      </Button>
                    )}
                  </Group>
                </form>
              )}

              {isOn && !editing && (
                <Group gap="sm" wrap="wrap">
                  <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    onClick={handleStartEditing}
                  >
                    Change TTL
                  </Button>
                  <Button
                    type="button"
                    color="red"
                    size="sm"
                    onClick={() =>
                      disableRuleMutation.mutate(
                        { path: { device_id: deviceId } },
                        {
                          onSuccess: () =>
                            notifications.show({
                              color: "green",
                              message: "Address lease rule disabled",
                            }),
                          onError: (err) =>
                            notifications.show({
                              color: "red",
                              title: "Error",
                              message: toErrorMessage(err),
                            }),
                        },
                      )
                    }
                    disabled={disableRuleMutation.isPending}
                  >
                    Turn off auto-expiry
                  </Button>
                </Group>
              )}
            </Stack>
          )}
        </Card>
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
