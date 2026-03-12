import { useEffect, useState } from "react";
import { useForm } from "@mantine/form";
import { zod4Resolver } from "mantine-form-zod-resolver";
import { z } from "zod";
import { format, isPast } from "date-fns";
import {
  Button,
  Card,
  Group,
  Modal,
  NativeSelect,
  Skeleton,
  Stack,
  Switch,
  Table,
  Text,
  TextInput,
  Title,
} from "@mantine/core";
import { notifications } from "@mantine/notifications";
import { toErrorMessage } from "@/lib/api-client";
import type { Address } from "@/lib/api";
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

const REFRESH_OPTIONS = [
  { label: "Off", value: 0 },
  { label: "1s", value: 1_000 },
  { label: "5s", value: 5_000 },
  { label: "15s", value: 15_000 },
  { label: "30s", value: 30_000 },
  { label: "1 min", value: 60_000 },
  { label: "5 min", value: 300_000 },
] as const;

const AUTO_HB_INTERVAL_OPTIONS = [
  { label: "30s", value: 30 },
  { label: "1m", value: 60 },
  { label: "5m", value: 300 },
  { label: "15m", value: 900 },
] as const;

const addressSchema = zAddAddressRequest;

interface DeviceAddressesTabProps {
  deviceId: number;
}

export function DeviceAddressesTab({ deviceId }: DeviceAddressesTabProps) {
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
  const [addressToDisable, setAddressToDisable] = useState<Address | null>(null);

  // Auto-heartbeat state
  const [ahSettings, setAhSettings] = useState(getAutoHeartbeatSettings);
  const [autoClientIp, setAutoClientIp] = useState<string | null>(getStoredClientIp);

  useEffect(() => {
    const onSettings = () => setAhSettings(getAutoHeartbeatSettings());
    const onClientIp = (e: Event) =>
      setAutoClientIp((e as CustomEvent<string>).detail);
    window.addEventListener(SETTINGS_EVENT, onSettings);
    window.addEventListener('storage', onSettings);
    window.addEventListener(CLIENT_IP_EVENT, onClientIp);
    return () => {
      window.removeEventListener(SETTINGS_EVENT, onSettings);
      window.removeEventListener('storage', onSettings);
      window.removeEventListener(CLIENT_IP_EVENT, onClientIp);
    };
  }, []);

  const isActive = ahSettings?.deviceId === deviceId;
  const currentInterval = isActive ? (ahSettings?.intervalSeconds ?? 60) : 60;

  function handleToggle(checked: boolean) {
    if (checked) {
      setAutoHeartbeatSettings({ deviceId, intervalSeconds: currentInterval });
    } else {
      clearAutoHeartbeatSettings();
    }
  }

  function handleIntervalChange(seconds: number) {
    setAutoHeartbeatSettings({ deviceId, intervalSeconds: seconds });
  }

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

  function handleConfirmDisable() {
    if (!addressToDisable) return;
    disableAddressMutation.mutate(
      { path: { device_id: deviceId, address_id: addressToDisable.id } },
      {
        onSuccess: () => notifications.show({ color: "green", message: "Address disabled" }),
        onError: (err) =>
          notifications.show({ color: "red", title: "Error disabling address", message: toErrorMessage(err) }),
        onSettled: () => setAddressToDisable(null),
      },
    );
  }

  return (
    <Stack gap="md">
      <Card withBorder>
        <Title order={4} mb="md">Heartbeat</Title>
        <Group gap="md">
          <Button
            type="button"
            onClick={() =>
              heartbeatMutation.mutate(
                { path: { device_id: deviceId } },
                {
                  onSuccess: (address) =>
                    notifications.show({ color: "green", message: `IP ${address.ip} registered` }),
                  onError: (err) =>
                    notifications.show({ color: "red", title: "Heartbeat failed", message: toErrorMessage(err) }),
                },
              )
            }
            disabled={heartbeatMutation.isPending}
          >
            {heartbeatMutation.isPending ? "Registering..." : "Register my IP"}
          </Button>
          {heartbeatMutation.data && (
            <Text size="sm" c="dimmed">
              Your current IP:{" "}
              <Text component="span" ff="monospace">{heartbeatMutation.data.ip}</Text>
            </Text>
          )}
        </Group>
      </Card>

      <Card withBorder>
        <Title order={4} mb="md">Keep browser IP registered</Title>
        <Stack gap="md">
          <Switch
            label="Automatically send heartbeat while this tab is open"
            checked={isActive}
            onChange={(event) => handleToggle(event.currentTarget.checked)}
          />
          {isActive && (
            <Group gap="lg">
              <Group gap="sm">
                <Text size="sm" c="dimmed" style={{ whiteSpace: "nowrap" }}>
                  Interval:
                </Text>
                <Group gap={4}>
                  {AUTO_HB_INTERVAL_OPTIONS.map((opt) => (
                    <button
                      key={opt.value}
                      type="button"
                      onClick={() => handleIntervalChange(opt.value)}
                      style={{
                        borderRadius: 4,
                        padding: "2px 8px",
                        fontSize: 12,
                        fontWeight: 500,
                        cursor: "pointer",
                        border: "none",
                        background: currentInterval === opt.value
                          ? "var(--mantine-color-blue-6)"
                          : "var(--mantine-color-default-border)",
                        color: currentInterval === opt.value
                          ? "#fff"
                          : "var(--mantine-color-dimmed)",
                      }}
                    >
                      {opt.label}
                    </button>
                  ))}
                </Group>
              </Group>
              {autoClientIp && (
                <Group gap={6}>
                  <span
                    style={{
                      display: "inline-block",
                      width: 8,
                      height: 8,
                      borderRadius: "50%",
                      background: "var(--mantine-color-green-6)",
                      flexShrink: 0,
                    }}
                  />
                  <Text size="sm" c="dimmed">
                    Your IP:{" "}
                    <Text component="span" ff="monospace">{autoClientIp}</Text>
                  </Text>
                </Group>
              )}
            </Group>
          )}
        </Stack>
      </Card>

      <Card withBorder>
        <Title order={4} mb="md">Add IP address</Title>
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
      </Card>

      <Card withBorder>
        <Group justify="space-between" mb="md">
          <Title order={4}>Assigned addresses</Title>
          <Group gap="sm">
            <Text size="sm" c="dimmed" style={{ whiteSpace: "nowrap" }}>
              Auto-refresh
            </Text>
            <NativeSelect
              value={refreshInterval}
              onChange={(e) => setRefreshInterval(Number(e.target.value))}
              data={REFRESH_OPTIONS.map((opt) => ({
                label: opt.label,
                value: String(opt.value),
              }))}
            />
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
                <Table.Th style={{ textAlign: "right" }}>Actions</Table.Th>
              </Table.Tr>
            </Table.Thead>
            <Table.Tbody>
              {addresses.map((address) => (
                <Table.Tr key={address.id}>
                  <Table.Td ff="monospace" fz="sm">{address.ip}</Table.Td>
                  <Table.Td>
                    <Group gap={8} title={address.is_enabled ? "Active" : "Inactive"}>
                      <span
                        style={{
                          display: "inline-block",
                          width: 10,
                          height: 10,
                          borderRadius: "50%",
                          flexShrink: 0,
                          background: address.is_enabled
                            ? "var(--mantine-color-green-6)"
                            : "var(--mantine-color-red-6)",
                        }}
                        aria-hidden
                      />
                      <Text size="sm" c="dimmed">
                        {address.is_enabled ? "Active" : "Inactive"}
                      </Text>
                    </Group>
                  </Table.Td>
                  <Table.Td>
                    <Text size="sm" c="dimmed">
                      {format(new Date(address.updated_at), "PP p")}
                    </Text>
                  </Table.Td>
                  <Table.Td>
                    {!address.is_enabled ? (
                      <Text size="sm" c="dimmed" style={{ opacity: 0.5 }}>Disabled address</Text>
                    ) : address.expires_at ? (
                      <Text
                        size="sm"
                        c={isPast(new Date(address.expires_at)) ? "red" : "dimmed"}
                      >
                        {format(new Date(address.expires_at), "PP p")}
                      </Text>
                    ) : (
                      <Text size="sm" c="dimmed" style={{ opacity: 0.5 }}>No expiry</Text>
                    )}
                  </Table.Td>
                  <Table.Td style={{ textAlign: "right" }}>
                    {address.is_enabled ? (
                      <Button
                        type="button"
                        color="red"
                        size="sm"
                        onClick={() => setAddressToDisable(address)}
                        disabled={disableAddressMutation.isPending}
                      >
                        Disable
                      </Button>
                    ) : (
                      <Button
                        type="button"
                        size="sm"
                        onClick={() =>
                          addAddressMutation.mutate(
                            { path: { device_id: deviceId }, body: { ip: address.ip } },
                            {
                              onSuccess: () => notifications.show({ color: "green", message: "Address added" }),
                              onError: (err) =>
                                notifications.show({ color: "red", title: "Error adding address", message: toErrorMessage(err) }),
                            },
                          )
                        }
                        disabled={addAddressMutation.isPending}
                      >
                        Enable
                      </Button>
                    )}
                  </Table.Td>
                </Table.Tr>
              ))}
            </Table.Tbody>
          </Table>
        )}
      </Card>

      <Modal
        opened={addressToDisable !== null}
        onClose={() => setAddressToDisable(null)}
        title="Disable address"
      >
        <Text size="sm">
          Disable IP{" "}
          <Text component="span" ff="monospace">{addressToDisable?.ip ?? ""}</Text>{" "}
          for this device? Existing connections may stop working.
        </Text>
        <Group justify="flex-end" mt="md" gap="sm">
          <Button type="button" variant="outline" onClick={() => setAddressToDisable(null)}>
            Cancel
          </Button>
          <Button
            type="button"
            color="red"
            onClick={handleConfirmDisable}
            disabled={disableAddressMutation.isPending}
          >
            {disableAddressMutation.isPending ? "Disabling..." : "Disable"}
          </Button>
        </Group>
      </Modal>
    </Stack>
  );
}
