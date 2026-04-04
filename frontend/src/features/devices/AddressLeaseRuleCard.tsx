import { useEffect, useState } from "react";
import { useForm, schemaResolver } from "@mantine/form";
import { z } from "zod";
import {
  Badge,
  Button,
  Card,
  Group,
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
import { zPutDeviceAddressLeaseRuleRequest } from "@/lib/api/zod.gen";

// ---------------------------------------------------------------------------
// TTL helpers
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// Form schema
// ---------------------------------------------------------------------------

type LeaseRuleFormValues = { value: string; unit: TtlUnit };

// value represents the human-readable count for the chosen unit (e.g. "5" for
// "5 minutes"). We borrow the gte(1) constraint from the generated ttl_seconds
// field: any value >= 1 with any unit satisfies ttl_seconds >= 1.
const leaseRuleFormSchema = z.object({
  value: z.preprocess(
    (v) => Number(v),
    zPutDeviceAddressLeaseRuleRequest.shape.ttl_seconds,
  ),
  unit: z.enum(TTL_UNITS),
});

// ---------------------------------------------------------------------------
// AddressLeaseRuleCard
// ---------------------------------------------------------------------------

export function AddressLeaseRuleCard({ deviceId }: { deviceId: number }) {
  const {
    data: addressLeaseRule,
    isLoading,
    isError,
    error,
  } = useDeviceAddressLeaseRule(deviceId);
  const putRuleMutation = usePutDeviceAddressLeaseRule(deviceId);
  const disableRuleMutation = useDisableDeviceAddressLeaseRule(deviceId);

  const form = useForm<LeaseRuleFormValues>({
    validate: schemaResolver(leaseRuleFormSchema),
    initialValues: { value: "5", unit: "minutes" },
  });
  const { setValues } = form;
  const [editing, setEditing] = useState(false);

  const isOn = Boolean(addressLeaseRule?.enabled);

  // When the rule is disabled, keep the form in sync with the server's last
  // known TTL so the user sees the current value if they re-enable.
  useEffect(() => {
    if (addressLeaseRule?.enabled) return;
    if (!addressLeaseRule) return;
    setValues(fromSeconds(addressLeaseRule.ttl_seconds));
  }, [addressLeaseRule, setValues]);

  function handleStartEditing() {
    if (!addressLeaseRule) return;
    setValues(fromSeconds(addressLeaseRule.ttl_seconds));
    setEditing(true);
  }

  function handleDisable() {
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
    );
  }

  function handleSubmit(values: LeaseRuleFormValues) {
    putRuleMutation.mutate(
      {
        path: { device_id: deviceId },
        body: { ttl_seconds: toSeconds(Number(values.value), values.unit) },
      },
      {
        onSuccess: () => {
          setEditing(false);
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

  const ttlLabel = addressLeaseRule?.ttl_seconds
    ? formatTtlLabel(addressLeaseRule.ttl_seconds)
    : null;

  return (
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
          {isOn ? (
            <Stack gap={4}>
              <Group gap="sm">
                <Text size="sm">Status:</Text>
                <Badge color="green" variant="light" size="sm">
                  Enabled
                </Badge>
              </Group>
              {ttlLabel && (
                <Group gap="sm">
                  <Text size="sm">TTL:</Text>
                  <Text size="sm" fw={600}>
                    {ttlLabel}
                  </Text>
                </Group>
              )}
            </Stack>
          ) : (
            <Group gap="sm">
              <Text size="sm">Status:</Text>
              <Badge color="red" variant="light" size="sm">
                Disabled
              </Badge>
              <Text size="sm" c="dimmed">
                Turn it on to automatically revoke stale addresses.
              </Text>
            </Group>
          )}
          {(!isOn || editing) && (
            <form onSubmit={form.onSubmit(handleSubmit)}>
              <Group align="flex-end" gap="md" wrap="wrap">
                <TextInput
                  label="Expires after"
                  type="number"
                  min={1}
                  step={1}
                  placeholder="1"
                  w={128}
                  {...form.getInputProps("value")}
                />
                <NativeSelect
                  label="Unit"
                  w={128}
                  data={TTL_UNITS.map((unit) => ({ label: unit, value: unit }))}
                  {...form.getInputProps("unit")}
                />
                <Button
                  type="submit"
                  disabled={putRuleMutation.isPending}
                  loading={putRuleMutation.isPending}
                >
                  {isOn ? "Save" : "Enable auto-expiry"}
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
                variant="outline"
                size="sm"
                onClick={handleDisable}
                disabled={disableRuleMutation.isPending}
                loading={disableRuleMutation.isPending}
              >
                Turn off auto-expiry
              </Button>
            </Group>
          )}
        </Stack>
      )}
    </Card>
  );
}
