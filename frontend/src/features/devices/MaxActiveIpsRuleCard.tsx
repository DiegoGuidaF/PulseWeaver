import { useEffect, useRef } from "react";
import { useForm, schemaResolver } from "@mantine/form";
import { z } from "zod";
import {
  ActionIcon,
  Badge,
  Button,
  Card,
  Divider,
  Group,
  NumberInput,
  Skeleton,
  Stack,
  Switch,
  Text,
  ThemeIcon,
  Tooltip,
} from "@mantine/core";
import { IconLayersSubtract, IconMinus, IconPlus } from "@tabler/icons-react";
import { notifications } from "@mantine/notifications";
import { toErrorMessage } from "@/lib/api-client";
import { useMaxActiveAddressesRule } from "@/features/devices/hooks/useMaxActiveAddressesRule";
import { usePutMaxActiveAddressesRule } from "@/features/devices/hooks/usePutMaxActiveAddressesRule";
import { useDisableMaxActiveAddressesRule } from "@/features/devices/hooks/useDisableMaxActiveAddressesRule";
import { zPutMaxActiveAddressesRuleRequest } from "@/lib/api/zod.gen";

const DEFAULT_LIMIT = 2;

type MaxAddressesFormValues = { max_addresses: string };

const maxAddressesFormSchema = z.object({
  max_addresses: z.preprocess(
    (v) => Number(v),
    zPutMaxActiveAddressesRuleRequest.shape.max_addresses,
  ),
});

interface MaxActiveIpsRuleCardProps {
  deviceId: number;
  liveAddressCount: number;
}

export function MaxActiveIpsRuleCard({ deviceId, liveAddressCount }: MaxActiveIpsRuleCardProps) {
  const {
    data: maxAddressesRule,
    isLoading,
    isError,
    error,
  } = useMaxActiveAddressesRule(deviceId);
  const putRuleMutation = usePutMaxActiveAddressesRule(deviceId);
  const disableRuleMutation = useDisableMaxActiveAddressesRule(deviceId);

  const form = useForm<MaxAddressesFormValues>({
    validate: schemaResolver(maxAddressesFormSchema),
    initialValues: { max_addresses: String(DEFAULT_LIMIT) },
  });
  const { setValues } = form;

  const isOn = Boolean(maxAddressesRule?.enabled);
  const savedLimit = maxAddressesRule?.max_addresses ?? null;
  const currentLimit = Number(form.values.max_addresses);
  const isDirty = isOn && savedLimit !== null && currentLimit !== savedLimit;
  const atLimit = isOn && savedLimit !== null && liveAddressCount >= savedLimit;
  const chipColor = isOn ? (atLimit ? "orange" : "teal") : "gray";
  const evictionCount = liveAddressCount > currentLimit ? liveAddressCount - currentLimit : 0;

  // Track the last server-side value we synced to avoid overwriting user edits,
  // while still syncing on first load and after a successful save.
  const syncedLimitRef = useRef<number | null>(null);
  useEffect(() => {
    if (!maxAddressesRule) return;
    if (maxAddressesRule.max_addresses == null) return;
    if (syncedLimitRef.current === maxAddressesRule.max_addresses) return;
    syncedLimitRef.current = maxAddressesRule.max_addresses;
    setValues({ max_addresses: String(maxAddressesRule.max_addresses) });
  }, [maxAddressesRule, setValues]);

  function handleToggleOn() {
    if (form.validate().hasErrors) return;
    putRuleMutation.mutate(
      { path: { device_id: deviceId }, body: { max_addresses: currentLimit } },
      {
        onSuccess: () =>
          notifications.show({ color: "green", message: "Max active IPs rule enabled" }),
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
          notifications.show({ color: "green", message: "Max active IPs rule disabled" }),
        onError: (err) =>
          notifications.show({ color: "red", title: "Error", message: toErrorMessage(err) }),
      },
    );
  }

  function handleSave() {
    if (form.validate().hasErrors) return;
    putRuleMutation.mutate(
      { path: { device_id: deviceId }, body: { max_addresses: currentLimit } },
      {
        onSuccess: () =>
          notifications.show({ color: "green", message: "Max active IPs rule saved" }),
        onError: (err) =>
          notifications.show({ color: "red", title: "Error", message: toErrorMessage(err) }),
      },
    );
  }

  function handleCancel() {
    if (!maxAddressesRule) return;
    setValues({ max_addresses: String(maxAddressesRule.max_addresses) });
  }

  function stepLimit(delta: number) {
    const next = Math.max(1, currentLimit + delta);
    setValues({ max_addresses: String(next) });
  }

  return (
    <Card withBorder>
      {/* Header */}
      <Group justify="space-between" align="flex-start">
        <Group gap="xs" align="flex-start">
          <ThemeIcon size="sm" variant="light" color={chipColor} mt={2}>
            <IconLayersSubtract size={12} />
          </ThemeIcon>
          <Stack gap={2}>
            <Text fw={600} size="sm">Max active IPs</Text>
            <Text size="xs" c="dimmed">
              Once exceeded, the oldest active address is evicted
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
            aria-label={isOn ? "Disable max-IP rule" : "Enable max-IP rule"}
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
            <Group gap="xs" align="center">
              <Text size="xs" c="dimmed" style={{ flexShrink: 0 }}>Limit</Text>
              <ActionIcon
                variant="default"
                size="sm"
                onClick={() => stepLimit(-1)}
                disabled={currentLimit <= 1}
                aria-label="Decrease limit"
              >
                <IconMinus size={12} />
              </ActionIcon>
              <NumberInput
                aria-label="Max active IPs"
                min={1}
                step={1}
                w={72}
                hideControls
                styles={{ input: { textAlign: "center" } }}
                {...form.getInputProps("max_addresses")}
              />
              <ActionIcon
                variant="default"
                size="sm"
                onClick={() => stepLimit(1)}
                aria-label="Increase limit"
              >
                <IconPlus size={12} />
              </ActionIcon>
              {isOn && savedLimit !== null && (
                <Tooltip
                  label={
                    atLimit
                      ? `Max active IPs · at limit (${liveAddressCount}/${savedLimit}) · next IP will evict oldest`
                      : `Max active IPs · ${liveAddressCount} of ${savedLimit}`
                  }
                  withArrow
                >
                  <Badge
                    color={chipColor}
                    variant={atLimit ? "filled" : "light"}
                    leftSection={<IconLayersSubtract size={10} stroke={1.5} />}
                    style={{ flexShrink: 0 }}
                  >
                    {liveAddressCount}/{savedLimit}
                  </Badge>
                </Tooltip>
              )}
            </Group>

            {atLimit && evictionCount === 0 && (
              <Text size="xs" c="orange">
                At limit · next new IP will evict the oldest active address
              </Text>
            )}
            {evictionCount > 0 && (
              <Text size="xs" c="orange">
                {evictionCount} active {evictionCount === 1 ? "address" : "addresses"} will be evicted when this limit is applied
              </Text>
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
