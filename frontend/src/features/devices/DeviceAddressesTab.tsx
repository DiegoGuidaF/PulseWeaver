import { useEffect, useState } from "react";
import { useForm } from "@mantine/form";
import { zod4Resolver } from "mantine-form-zod-resolver";
import { z } from "zod";
import { isPast } from "@/lib/dates";
import { useDateFormatter } from "@/contexts/useDateTimePrefs";
import {
  ActionIcon,
  Button,
  Card,
  Group,
  SegmentedControl,
  Skeleton,
  Stack,
  Switch,
  Table,
  Text,
  TextInput,
  Title,
  Tooltip,
} from "@mantine/core";
import { AutoRefreshSelect } from "@/components/AutoRefreshSelect";
import { IconPlayerPlay, IconPlayerStop } from "@tabler/icons-react";
import { notifications } from "@mantine/notifications";
import { toErrorMessage } from "@/lib/api-client";
import { zAddAddressRequest } from "@/lib/api/zod.gen";
import { useDeviceAddresses } from "@/features/devices/hooks/useDeviceAddresses";
import { useAddDeviceAddress } from "@/features/devices/hooks/useAddDeviceAddress";
import { useDisableDeviceAddress } from "@/features/devices/hooks/useDisableDeviceAddress";
import { useDeviceHeartbeat } from "@/features/devices/hooks/useDeviceHeartbeat";
import {
  getAutoHeartbeatSettings,
  setAutoHeartbeatSettings,
  clearAutoHeartbeatSettings,
  getStoredClientIp,
  CLIENT_IP_EVENT,
  SETTINGS_EVENT,
} from "@/lib/autoHeartbeat";

const AUTO_HB_INTERVAL_OPTIONS = [
  { label: "30s", value: 30 },
  { label: "1m", value: 60 },
  { label: "5m", value: 300 },
  { label: "15m", value: 900 },
] as const;

const addressSchema = zAddAddressRequest;

const RegisterMode = {
  MyIp: "my-ip",
  Custom: "custom",
} as const;
type RegisterMode = (typeof RegisterMode)[keyof typeof RegisterMode];

function StatusDot({ color, size = 8 }: { color: string; size?: number }) {
  return (
    <span
      aria-hidden
      style={{
        display: "inline-block",
        width: size,
        height: size,
        borderRadius: "50%",
        flexShrink: 0,
        background: `var(--mantine-color-${color}-6)`,
      }}
    />
  );
}

function useAutoHeartbeat(deviceId: number) {
  const [settings, setSettings] = useState(getAutoHeartbeatSettings);
  const [clientIp, setClientIp] = useState<string | null>(getStoredClientIp);

  useEffect(() => {
    const onSettings = () => setSettings(getAutoHeartbeatSettings());
    const onClientIp = (e: Event) => setClientIp((e as CustomEvent<string>).detail);
    window.addEventListener(SETTINGS_EVENT, onSettings);
    window.addEventListener("storage", onSettings);
    window.addEventListener(CLIENT_IP_EVENT, onClientIp);
    return () => {
      window.removeEventListener(SETTINGS_EVENT, onSettings);
      window.removeEventListener("storage", onSettings);
      window.removeEventListener(CLIENT_IP_EVENT, onClientIp);
    };
  }, []);

  const isActive = settings?.deviceId === deviceId;
  const intervalSeconds = isActive ? (settings?.intervalSeconds ?? 60) : 60;

  function toggle(checked: boolean) {
    if (checked) {
      setAutoHeartbeatSettings({ deviceId, intervalSeconds });
    } else {
      clearAutoHeartbeatSettings();
    }
  }

  function changeInterval(seconds: number) {
    setAutoHeartbeatSettings({ deviceId, intervalSeconds: seconds });
  }

  return { isActive, intervalSeconds, clientIp, toggle, changeInterval };
}

interface DeviceAddressesTabProps {
  deviceId: number;
}

export function DeviceAddressesTab({ deviceId }: DeviceAddressesTabProps) {
  const formatDateTime = useDateFormatter();
  const [refreshInterval, setRefreshInterval] = useState<number>(5_000);
  const { data: addresses, isLoading } = useDeviceAddresses(
    deviceId,
    true,
    refreshInterval === 0 ? false : refreshInterval,
  );
  const heartbeatMutation = useDeviceHeartbeat();
  const form = useForm<z.infer<typeof addressSchema>>({
    validate: zod4Resolver(addressSchema),
    initialValues: { ip: "" },
  });
  const addAddressMutation = useAddDeviceAddress({
    onSuccess: () => form.reset(),
  });
  const disableAddressMutation = useDisableDeviceAddress();

  const [registerMode, setRegisterMode] = useState<RegisterMode>(RegisterMode.MyIp);
  const autoHeartbeat = useAutoHeartbeat(deviceId);

  function handleAddAddressSubmit(values: z.infer<typeof addressSchema>) {
    addAddressMutation.mutate(
      { path: { device_id: deviceId }, body: { ip: values.ip } },
      {
        onSuccess: () => notifications.show({ color: "green", message: "Address added" }),
        onError: (err) =>
          notifications.show({ color: "red", title: "Error adding address", message: toErrorMessage(err) }),
      },
    );
  }

  function handleDisable(addressId: number) {
    disableAddressMutation.mutate(
      { path: { device_id: deviceId, address_id: addressId } },
      {
        onSuccess: () => notifications.show({ color: "green", message: "Address disabled" }),
        onError: (err) =>
          notifications.show({ color: "red", title: "Error disabling address", message: toErrorMessage(err) }),
      },
    );
  }

  function handleHeartbeat() {
    heartbeatMutation.mutate(
      { path: { device_id: deviceId } },
      {
        onSuccess: (address) =>
          notifications.show({ color: "green", message: `IP ${address.ip} registered` }),
        onError: (err) =>
          notifications.show({ color: "red", title: "Heartbeat failed", message: toErrorMessage(err) }),
      },
    );
  }

  function handleReEnable(ip: string) {
    addAddressMutation.mutate(
      { path: { device_id: deviceId }, body: { ip } },
      {
        onSuccess: () => notifications.show({ color: "green", message: "Address enabled" }),
        onError: (err) =>
          notifications.show({ color: "red", title: "Error", message: toErrorMessage(err) }),
      },
    );
  }

  return (
    <Stack gap="md">
      {/* Register IP address — unified card */}
      <Card withBorder>
        <Title order={4} mb="md">Register IP address</Title>
        <Stack gap="md">
          <SegmentedControl
            size="xs"
            value={registerMode}
            onChange={(v) => setRegisterMode(v as RegisterMode)}
            data={[
              { label: "My current IP", value: RegisterMode.MyIp },
              { label: "Custom IP", value: RegisterMode.Custom },
            ]}
          />

          {registerMode === RegisterMode.MyIp ? (
            <Stack gap="md">
              <Group gap="md">
                <Button
                  type="button"
                  onClick={handleHeartbeat}
                  disabled={heartbeatMutation.isPending}
                >
                  {heartbeatMutation.isPending ? "Registering..." : "Register my IP"}
                </Button>
                {heartbeatMutation.data && (
                  <Text size="sm" c="dimmed">
                    Your IP:{" "}
                    <Text component="span" ff="monospace">{heartbeatMutation.data.ip}</Text>
                  </Text>
                )}
              </Group>

              <Switch
                label="Auto-register while this tab is open"
                checked={autoHeartbeat.isActive}
                onChange={(event) => autoHeartbeat.toggle(event.currentTarget.checked)}
              />
              {autoHeartbeat.isActive && (
                <Group gap="lg">
                  <Group gap="sm">
                    <Text size="sm" c="dimmed" style={{ whiteSpace: "nowrap" }}>
                      Interval:
                    </Text>
                    <SegmentedControl
                      size="xs"
                      value={String(autoHeartbeat.intervalSeconds)}
                      onChange={(v) => autoHeartbeat.changeInterval(Number(v))}
                      data={AUTO_HB_INTERVAL_OPTIONS.map((opt) => ({
                        label: opt.label,
                        value: String(opt.value),
                      }))}
                    />
                  </Group>
                  {autoHeartbeat.clientIp && (
                    <Group gap={6}>
                      <StatusDot color="green" />
                      <Text size="sm" c="dimmed">
                        IP:{" "}
                        <Text component="span" ff="monospace">{autoHeartbeat.clientIp}</Text>
                      </Text>
                    </Group>
                  )}
                </Group>
              )}
            </Stack>
          ) : (
            <form onSubmit={form.onSubmit(handleAddAddressSubmit)}>
              <Group align="flex-end" gap="md">
                <TextInput
                  label="IP address"
                  placeholder="192.168.1.100"
                  autoComplete="off"
                  style={{ flex: 1 }}
                  {...form.getInputProps("ip")}
                />
                <Button type="submit" disabled={addAddressMutation.isPending}>
                  {addAddressMutation.isPending ? "Adding..." : "Add IP"}
                </Button>
              </Group>
            </form>
          )}
        </Stack>
      </Card>

      {/* Assigned addresses */}
      <Card withBorder>
        <Group justify="space-between" mb="md">
          <Title order={4}>Assigned addresses</Title>
          <Group gap="xs">
            <Text size="sm" c="dimmed">Refresh:</Text>
            <AutoRefreshSelect value={refreshInterval} onChange={setRefreshInterval} />
          </Group>
        </Group>

        {isLoading ? (
          <Stack gap={8}>
            <Skeleton height={16} />
            <Skeleton height={16} />
            <Skeleton height={16} width="66%" />
          </Stack>
        ) : !addresses || addresses.length === 0 ? (
          <Text size="sm" c="dimmed">No addresses assigned yet.</Text>
        ) : (
          <Table>
            <Table.Thead>
              <Table.Tr>
                <Table.Th>IP</Table.Th>
                <Table.Th>Status</Table.Th>
                <Table.Th>Updated</Table.Th>
                <Table.Th>Expires</Table.Th>
                <Table.Th w={48} />
              </Table.Tr>
            </Table.Thead>
            <Table.Tbody>
              {addresses.map((address) => (
                <Table.Tr key={address.id}>
                  <Table.Td ff="monospace" fz="sm">{address.ip}</Table.Td>
                  <Table.Td>
                    <Group gap={8} title={address.is_enabled ? "Active" : "Inactive"}>
                      <StatusDot color={address.is_enabled ? "green" : "red"} size={10} />
                      <Text size="sm" c="dimmed">
                        {address.is_enabled ? "Active" : "Inactive"}
                      </Text>
                    </Group>
                  </Table.Td>
                  <Table.Td>
                    <Text size="sm" c="dimmed">
                      {formatDateTime(address.updated_at)}
                    </Text>
                  </Table.Td>
                  <Table.Td>
                    {address.expires_at && address.is_enabled ? (
                      <Text
                        size="sm"
                        c={isPast(address.expires_at) ? "red" : "dimmed"}
                      >
                        {formatDateTime(address.expires_at)}
                      </Text>
                    ) : (
                      <Text size="sm" c="dimmed" style={{ opacity: 0.5 }}>No expiry</Text>
                    )}
                  </Table.Td>
                  <Table.Td>
                    {address.is_enabled ? (
                      <Tooltip label="Disable address" withArrow>
                        <ActionIcon
                          variant="subtle"
                          color="red"
                          onClick={() => handleDisable(address.id)}
                          disabled={disableAddressMutation.isPending}
                          aria-label="Disable address"
                        >
                          <IconPlayerStop size={16} stroke={1.5} />
                        </ActionIcon>
                      </Tooltip>
                    ) : (
                      <Tooltip label="Re-enable address" withArrow>
                        <ActionIcon
                          variant="subtle"
                          color="green"
                          onClick={() => handleReEnable(address.ip)}
                          disabled={addAddressMutation.isPending}
                          aria-label="Enable address"
                        >
                          <IconPlayerPlay size={16} stroke={1.5} />
                        </ActionIcon>
                      </Tooltip>
                    )}
                  </Table.Td>
                </Table.Tr>
              ))}
            </Table.Tbody>
          </Table>
        )}
      </Card>
    </Stack>
  );
}
