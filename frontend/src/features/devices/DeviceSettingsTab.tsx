import { useEffect, useState } from "react";
import { useForm, schemaResolver } from "@mantine/form";
import { z } from "zod";
import {
  Badge,
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
import { useMaxActiveAddressesRule } from "@/features/devices/hooks/useMaxActiveAddressesRule";
import { usePutMaxActiveAddressesRule } from "@/features/devices/hooks/usePutMaxActiveAddressesRule";
import { useDisableMaxActiveAddressesRule } from "@/features/devices/hooks/useDisableMaxActiveAddressesRule";

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

type MaxAddressesFormValues = { max_addresses: string };

const maxAddressesFormSchema = z.object({
  max_addresses: z.coerce
    .number()
    .int("Must be a whole number")
    .min(1, "Minimum is 1"),
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
    data: addressLeaseRule,
    isLoading: isAddressLeaseLoading,
    isError: isAddressLeaseError,
    error: addressLeaseError,
  } = useDeviceAddressLeaseRule(deviceId);
  const putRuleMutation = usePutDeviceAddressLeaseRule(deviceId);
  const disableRuleMutation = useDisableDeviceAddressLeaseRule(deviceId);
  const regenerateApiKey = useRegenerateApiKey();

  const {
    data: maxAddressesRule,
    isLoading: isMaxAddressesLoading,
    isError: isMaxAddressesError,
    error: maxAddressesError,
  } = useMaxActiveAddressesRule(deviceId);
  const putMaxAddressesRuleMutation = usePutMaxActiveAddressesRule(deviceId);
  const disableMaxAddressesRuleMutation =
    useDisableMaxActiveAddressesRule(deviceId);

  const [regeneratedApiKey, setRegeneratedApiKey] = useState<string | null>(
    null,
  );
  const [confirmRegenOpen, setConfirmRegenOpen] = useState(false);

  const leaseRuleForm = useForm<LeaseRuleFormValues>({
    validate: schemaResolver(leaseRuleFormSchema),
    initialValues: { value: "5", unit: "minutes" },
  });
  const { setValues: setAddressLeaseValues } = leaseRuleForm;
  const [addressLeaseEditing, setAddressLeaseEditing] = useState(false);

  const isAddressLeaseOn = Boolean(addressLeaseRule && addressLeaseRule.enabled);

  const maxAddressesForm = useForm<MaxAddressesFormValues>({
    validate: schemaResolver(maxAddressesFormSchema),
    initialValues: { max_addresses: "2" },
  });
  const { setValues: setMaxAddressesValues } = maxAddressesForm;
  const [maxAddressesEditing, setMaxAddressesEditing] = useState(false);

  const isMaxAddressesOn = Boolean(
    maxAddressesRule && maxAddressesRule.enabled,
  );

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

  function handleAddressLeaseSubmit(values: LeaseRuleFormValues) {
    putRuleMutation.mutate(
      {
        path: { device_id: deviceId },
        body: { ttl_seconds: toSeconds(Number(values.value), values.unit) },
      },
      {
        onSuccess: () => {
          setAddressLeaseEditing(false);
          notifications.show({
            color: "green",
            message: "Address lease rule saved",
          });
        },
        onError: (err) =>
          notifications.show({
            color: "red",
            title: "Error",
            message: toErrorMessage(err),
          }),
      },
    );
  }

  function handleAddressLeaseStartEditing() {
    if (!addressLeaseRule) return;
    setAddressLeaseValues(fromSeconds(addressLeaseRule.ttl_seconds));
    setAddressLeaseEditing(true);
  }

  useEffect(() => {
    if (!addressLeaseRule || isAddressLeaseOn) return;
    setAddressLeaseValues(fromSeconds(addressLeaseRule.ttl_seconds));
  }, [isAddressLeaseOn, addressLeaseRule, setAddressLeaseValues]);

  useEffect(() => {
    if (!maxAddressesRule || isMaxAddressesOn) return;
    setMaxAddressesValues({
      max_addresses: String(maxAddressesRule.max_addresses),
    });
  }, [isMaxAddressesOn, maxAddressesRule, setMaxAddressesValues]);

  function handleMaxAddressesSubmit(values: MaxAddressesFormValues) {
    putMaxAddressesRuleMutation.mutate(
      {
        path: { device_id: deviceId },
        body: { max_addresses: Number(values.max_addresses) },
      },
      {
        onSuccess: () => {
          setMaxAddressesEditing(false);
          notifications.show({
            color: "green",
            message: "Max active IPs rule saved",
          });
        },
        onError: (err) =>
          notifications.show({
            color: "red",
            title: "Error",
            message: toErrorMessage(err),
          }),
      },
    );
  }

  function handleStartEditingMaxAddresses() {
    if (!maxAddressesRule) return;
    setMaxAddressesValues({
      max_addresses: String(maxAddressesRule.max_addresses),
    });
    setMaxAddressesEditing(true);
  }

  const ttlLabel =
    addressLeaseRule && addressLeaseRule.ttl_seconds ? formatTtlLabel(addressLeaseRule.ttl_seconds) : null;
  const submitButtonLabel = putRuleMutation.isPending
    ? "Saving..."
    : isAddressLeaseOn
      ? "Save"
      : "Enable auto-expiry";

  const maxAddressesSubmitLabel = putMaxAddressesRuleMutation.isPending
    ? "Saving..."
    : isMaxAddressesOn
      ? "Save"
      : "Enable max-IP rule";

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
          {isAddressLeaseLoading ? (
            <Stack gap={8}>
              <Skeleton height={16} width={160} />
              <Skeleton height={16} width={256} />
            </Stack>
          ) : isAddressLeaseError ? (
            <Text size="sm" c="red">
              Error loading rule: {toErrorMessage(addressLeaseError)}
            </Text>
          ) : (
            <Stack gap="md">
              {isAddressLeaseOn && (
                <Stack gap={4}>
                  <Group gap="sm">
                    <Text size="sm">Status:</Text>
                    <Badge color="green" variant="light" size="sm">Enabled</Badge>
                  </Group>
                  {ttlLabel && (
                    <Group gap="sm">
                      <Text size="sm">TTL:</Text>
                      <Text size="sm" fw={600}>{ttlLabel}</Text>
                    </Group>
                  )}
                </Stack>
              )}

              {!isAddressLeaseOn && (
                <Group gap="sm">
                  <Text size="sm">Status:</Text>
                  <Badge color="red" variant="light" size="sm">Disabled</Badge>
                  <Text size="sm" c="dimmed">
                    Turn it on to automatically revoke stale addresses.
                  </Text>
                </Group>
              )}

              {(!isAddressLeaseOn || addressLeaseEditing) && (
                <form onSubmit={leaseRuleForm.onSubmit(handleAddressLeaseSubmit)}>
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
                    {addressLeaseEditing && (
                      <Button
                        type="button"
                        variant="outline"
                        onClick={() => setAddressLeaseEditing(false)}
                      >
                        Cancel
                      </Button>
                    )}
                  </Group>
                </form>
              )}

              {isAddressLeaseOn && !addressLeaseEditing && (
                <Group gap="sm" wrap="wrap">
                  <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    onClick={handleAddressLeaseStartEditing}
                  >
                    Change TTL
                  </Button>
                  <Button
                    type="button"
                    variant="outline"
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
        <Card withBorder>
          <Title order={4} mb="md">
            Max active IPs rule
          </Title>
          {isMaxAddressesLoading ? (
            <Stack gap={8}>
              <Skeleton height={16} width={160} />
              <Skeleton height={16} width={256} />
            </Stack>
          ) : isMaxAddressesError ? (
            <Text size="sm" c="red">
              Error loading rule: {toErrorMessage(maxAddressesError)}
            </Text>
          ) : (
            <Stack gap="md">
              {isMaxAddressesOn && (
                <Stack gap={4}>
                  <Group gap="sm">
                    <Text size="sm">Status:</Text>
                    <Badge color="green" variant="light" size="sm">
                      Enabled
                    </Badge>
                  </Group>
                  <Group gap="sm">
                    <Text size="sm">Max IPs:</Text>
                    <Text size="sm" fw={600}>
                      {maxAddressesRule!.max_addresses}
                    </Text>
                  </Group>
                </Stack>
              )}

              {!isMaxAddressesOn && (
                <Group gap="sm">
                  <Text size="sm">Status:</Text>
                  <Badge color="red" variant="light" size="sm">
                    Disabled
                  </Badge>
                  <Text size="sm" c="dimmed">
                    Turn it on to limit the number of simultaneously active IPs.
                  </Text>
                </Group>
              )}

              {(!isMaxAddressesOn || maxAddressesEditing) && (
                <form
                  onSubmit={maxAddressesForm.onSubmit(
                    handleMaxAddressesSubmit,
                  )}
                >
                  <Group align="flex-end" gap="md" wrap="wrap">
                    <TextInput
                      label="Max active IPs"
                      type="number"
                      min={1}
                      step={1}
                      placeholder="3"
                      w={128}
                      {...maxAddressesForm.getInputProps("max_addresses")}
                    />
                    <Button
                      type="submit"
                      disabled={putMaxAddressesRuleMutation.isPending}
                    >
                      {maxAddressesSubmitLabel}
                    </Button>
                    {maxAddressesEditing && (
                      <Button
                        type="button"
                        variant="outline"
                        onClick={() => setMaxAddressesEditing(false)}
                      >
                        Cancel
                      </Button>
                    )}
                  </Group>
                </form>
              )}

              {isMaxAddressesOn && !maxAddressesEditing && (
                <Group gap="sm" wrap="wrap">
                  <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    onClick={handleStartEditingMaxAddresses}
                  >
                    Change limit
                  </Button>
                  <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    onClick={() =>
                      disableMaxAddressesRuleMutation.mutate(
                        { path: { device_id: deviceId } },
                        {
                          onSuccess: () =>
                            notifications.show({
                              color: "green",
                              message: "Max active IPs rule disabled",
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
                    disabled={disableMaxAddressesRuleMutation.isPending}
                  >
                    Turn off max-IP rule
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
