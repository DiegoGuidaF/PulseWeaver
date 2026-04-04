import { useEffect, useState } from "react";
import { useForm, schemaResolver } from "@mantine/form";
import { z } from "zod";
import {
  Badge,
  Button,
  Card,
  Group,
  Skeleton,
  Stack,
  Text,
  TextInput,
  Title,
} from "@mantine/core";
import { notifications } from "@mantine/notifications";
import { toErrorMessage } from "@/lib/api-client";
import { useMaxActiveAddressesRule } from "@/features/devices/hooks/useMaxActiveAddressesRule";
import { usePutMaxActiveAddressesRule } from "@/features/devices/hooks/usePutMaxActiveAddressesRule";
import { useDisableMaxActiveAddressesRule } from "@/features/devices/hooks/useDisableMaxActiveAddressesRule";
import { zPutMaxActiveAddressesRuleRequest } from "@/lib/api/zod.gen";

// ---------------------------------------------------------------------------
// Form schema
// ---------------------------------------------------------------------------

type MaxAddressesFormValues = { max_addresses: string };

// The form stores max_addresses as a string (HTML number input); preprocess
// coerces it to a number before the generated z.int().gte(1) constraint runs.
const maxAddressesFormSchema = z.object({
  max_addresses: z.preprocess(
    (v) => Number(v),
    zPutMaxActiveAddressesRuleRequest.shape.max_addresses,
  ),
});

// ---------------------------------------------------------------------------
// MaxActiveIpsRuleCard
// ---------------------------------------------------------------------------

export function MaxActiveIpsRuleCard({ deviceId }: { deviceId: number }) {
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
    initialValues: { max_addresses: "2" },
  });
  const { setValues } = form;
  const [editing, setEditing] = useState(false);

  const isOn = Boolean(maxAddressesRule?.enabled);

  // When the rule is disabled, keep the form in sync with the server's last
  // known limit so the user sees the current value if they re-enable.
  useEffect(() => {
    if (maxAddressesRule?.enabled) return;
    if (!maxAddressesRule) return;
    setValues({ max_addresses: String(maxAddressesRule.max_addresses) });
  }, [maxAddressesRule, setValues]);

  function handleStartEditing() {
    if (!maxAddressesRule) return;
    setValues({ max_addresses: String(maxAddressesRule.max_addresses) });
    setEditing(true);
  }

  function handleDisable() {
    disableRuleMutation.mutate(
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
    );
  }

  function handleSubmit(values: MaxAddressesFormValues) {
    putRuleMutation.mutate(
      {
        path: { device_id: deviceId },
        body: { max_addresses: Number(values.max_addresses) },
      },
      {
        onSuccess: () => {
          setEditing(false);
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

  return (
    <Card withBorder>
      <Title order={4} mb="md">
        Max active IPs rule
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
              <Group gap="sm">
                <Text size="sm">Max IPs:</Text>
                <Text size="sm" fw={600}>
                  {maxAddressesRule?.max_addresses}
                </Text>
              </Group>
            </Stack>
          ) : (
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
          {(!isOn || editing) && (
            <form onSubmit={form.onSubmit(handleSubmit)}>
              <Group align="flex-end" gap="md" wrap="wrap">
                <TextInput
                  label="Max active IPs"
                  type="number"
                  min={1}
                  step={1}
                  placeholder="3"
                  w={128}
                  {...form.getInputProps("max_addresses")}
                />
                <Button
                  type="submit"
                  disabled={putRuleMutation.isPending}
                  loading={putRuleMutation.isPending}
                >
                  {isOn ? "Save" : "Enable max-IP rule"}
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
                Change limit
              </Button>
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={handleDisable}
                disabled={disableRuleMutation.isPending}
                loading={disableRuleMutation.isPending}
              >
                Turn off max-IP rule
              </Button>
            </Group>
          )}
        </Stack>
      )}
    </Card>
  );
}
