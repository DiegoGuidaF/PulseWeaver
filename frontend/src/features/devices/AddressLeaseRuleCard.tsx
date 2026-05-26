import { useEffect } from "react";
import { useForm, schemaResolver } from "@mantine/form";
import { z } from "zod";
import {
  Badge,
  Button,
  Card,
  Divider,
  Group,
  NativeSelect,
  NumberInput,
  SegmentedControl,
  Skeleton,
  Stack,
  Switch,
  Text,
  ThemeIcon,
  Tooltip,
} from "@mantine/core";
import { IconClock } from "@tabler/icons-react";
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

const TTL_PRESETS = [
  { label: "30s", value: "30" },
  { label: "5m", value: "300" },
  { label: "15m", value: "900" },
  { label: "1h", value: "3600" },
  { label: "6h", value: "21600" },
  { label: "24h", value: "86400" },
  { label: "Custom…", value: "custom" },
] as const;

const PRESET_VALUES = new Set<string>(TTL_PRESETS.filter((p) => p.value !== "custom").map((p) => p.value));

const DEFAULT_TTL = 3600;

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

function formatTtlBadge(ttlSeconds: number): string {
  if (ttlSeconds < 60) return `${ttlSeconds}s`;
  if (ttlSeconds < 3600) return `${Math.round(ttlSeconds / 60)}m`;
  if (ttlSeconds < SECONDS_PER_DAY) return `${Math.round(ttlSeconds / 3600)}h`;
  return `${Math.round(ttlSeconds / SECONDS_PER_DAY)}d`;
}

function presetFromTtl(ttlSeconds: number): string {
  const key = String(ttlSeconds);
  return PRESET_VALUES.has(key) ? key : "custom";
}

// ---------------------------------------------------------------------------
// Form schema
// ---------------------------------------------------------------------------

type LeaseRuleFormValues = { value: string; unit: TtlUnit; preset: string };

const leaseRuleFormSchema = z.object({
  value: z.preprocess(
    (v) => Number(v),
    zPutDeviceAddressLeaseRuleRequest.shape.ttl_seconds,
  ),
  unit: z.enum(TTL_UNITS),
  preset: z.string(),
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
    initialValues: { value: "1", unit: "hours", preset: String(DEFAULT_TTL) },
  });
  const { setValues } = form;

  const isOn = Boolean(addressLeaseRule?.enabled);
  const preset = form.values.preset;

  // Dirty only matters when enabled: controls shown to change the active value
  const savedTtl = addressLeaseRule?.ttl_seconds;
  const currentTtl =
    preset === "custom"
      ? toSeconds(Number(form.values.value), form.values.unit)
      : Number(preset);
  const isDirty = isOn && savedTtl !== undefined && currentTtl !== savedTtl;

  useEffect(() => {
    if (!addressLeaseRule || isDirty) return;
    const ttl = addressLeaseRule.ttl_seconds ?? DEFAULT_TTL;
    setValues({ ...fromSeconds(ttl), preset: presetFromTtl(ttl) });
  }, [addressLeaseRule, setValues, isDirty]);

  function handleToggleOn() {
    if (preset === "custom" && form.validate().hasErrors) return;
    const ttlSeconds =
      preset === "custom"
        ? toSeconds(Number(form.values.value), form.values.unit)
        : Number(preset);
    putRuleMutation.mutate(
      { path: { device_id: deviceId }, body: { ttl_seconds: ttlSeconds } },
      {
        onSuccess: () =>
          notifications.show({ color: "green", message: "Address lease rule enabled" }),
        onError: (err) =>
          notifications.show({ color: "red", title: "Error", message: toErrorMessage(err) }),
      },
    );
  }

  function handleToggleOff() {
    disableRuleMutation.mutate(
      { path: { device_id: deviceId } },
      {
        onSuccess: () =>
          notifications.show({ color: "green", message: "Address lease rule disabled" }),
        onError: (err) =>
          notifications.show({ color: "red", title: "Error", message: toErrorMessage(err) }),
      },
    );
  }

  function handleSave() {
    if (preset === "custom" && form.validate().hasErrors) return;
    putRuleMutation.mutate(
      { path: { device_id: deviceId }, body: { ttl_seconds: currentTtl } },
      {
        onSuccess: () =>
          notifications.show({ color: "green", message: "Address lease rule saved" }),
        onError: (err) =>
          notifications.show({ color: "red", title: "Error", message: toErrorMessage(err) }),
      },
    );
  }

  function handleCancel() {
    if (!addressLeaseRule) return;
    const ttl = addressLeaseRule.ttl_seconds ?? DEFAULT_TTL;
    setValues({ ...fromSeconds(ttl), preset: presetFromTtl(ttl) });
  }

  function handlePresetChange(value: string) {
    if (value !== "custom") {
      setValues({ ...fromSeconds(Number(value)), preset: value });
    } else {
      form.setFieldValue("preset", "custom");
    }
  }

  return (
    <Card withBorder>
      {/* Header */}
      <Group justify="space-between" align="flex-start">
        <Group gap="xs" align="flex-start">
          <ThemeIcon size="sm" variant="light" color={isOn ? "teal" : "gray"} mt={2}>
            <IconClock size={12} />
          </ThemeIcon>
          <Stack gap={2}>
            <Text fw={600} size="sm">Auto-expiry</Text>
            <Text size="xs" c="dimmed">
              Active addresses expire after this long without activity
            </Text>
          </Stack>
        </Group>
        {isLoading ? (
          <Skeleton height={20} width={72} />
        ) : (
          <Switch
            size="sm"
            checked={isOn}
            disabled={putRuleMutation.isPending || disableRuleMutation.isPending}
            onChange={(e) => {
              if (e.currentTarget.checked) handleToggleOn();
              else handleToggleOff();
            }}
            label={isOn ? "Enabled" : "Disabled"}
            aria-label={isOn ? "Disable auto-expiry" : "Enable auto-expiry"}
          />
        )}
      </Group>

      {isLoading ? (
        <Stack gap={8} mt="md">
          <Skeleton height={16} width={160} />
          <Skeleton height={16} width={256} />
        </Stack>
      ) : isError ? (
        <Text size="sm" c="red" mt="md">
          Error loading rule: {toErrorMessage(error)}
        </Text>
      ) : (
        <>
          <Divider my="sm" />
          {/* Controls always visible; dimmed when disabled so the user can
              preview and adjust the value before enabling */}
          <Stack gap="sm" style={{ opacity: isOn ? 1 : 0.5 }}>
            <Group gap="sm" align="center" wrap="nowrap">
              <Text size="xs" c="dimmed" style={{ flexShrink: 0 }}>TTL</Text>
              <SegmentedControl
                size="xs"
                value={preset}
                onChange={handlePresetChange}
                data={TTL_PRESETS.map((p) => ({ label: p.label, value: p.value }))}
                style={{ width: "fit-content" }}
              />
              {isOn && addressLeaseRule?.ttl_seconds != null && (
                <Tooltip label={`Auto-expiry · TTL ${formatTtlBadge(addressLeaseRule.ttl_seconds)}`} withArrow>
                  <Badge color="teal" variant="light" leftSection={<IconClock size={10} stroke={1.5} />} style={{ flexShrink: 0 }}>
                    {formatTtlBadge(addressLeaseRule.ttl_seconds)}
                  </Badge>
                </Tooltip>
              )}
            </Group>

            {preset === "custom" && (
              <Group align="flex-end" gap="sm" wrap="wrap">
                <NumberInput
                  label="Value"
                  min={1}
                  step={1}
                  placeholder="1"
                  w={100}
                  {...form.getInputProps("value")}
                />
                <NativeSelect
                  label="Unit"
                  w={120}
                  data={TTL_UNITS.map((unit) => ({ label: unit, value: unit }))}
                  {...form.getInputProps("unit")}
                />
              </Group>
            )}

            {isDirty && (
              <Group gap="sm">
                <Button
                  size="xs"
                  onClick={handleSave}
                  disabled={putRuleMutation.isPending}
                  loading={putRuleMutation.isPending}
                >
                  Save
                </Button>
                <Button size="xs" variant="subtle" onClick={handleCancel}>
                  Cancel
                </Button>
              </Group>
            )}
          </Stack>
        </>
      )}
    </Card>
  );
}
