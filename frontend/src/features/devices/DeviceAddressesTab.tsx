import { useEffect, useMemo, useRef, useState } from "react";
import dayjs from "dayjs";
import relativeTime from "dayjs/plugin/relativeTime";
import {
  ActionIcon,
  Alert,
  Anchor,
  Box,
  Button,
  Code,
  Collapse,
  Group,
  Progress,
  SegmentedControl,
  Skeleton,
  Stack,
  Switch,
  Table,
  Text,
  TextInput,
  Tooltip,
} from "@mantine/core";
import { IconAlertTriangle, IconSearch, IconX } from "@tabler/icons-react";
import { notifications } from "@mantine/notifications";
import { toErrorMessage } from "@/lib/api-client";
import { ErrorState } from "@/components/ErrorState";
import { useDateFormatter } from "@/contexts/useDateTimePrefs";
import { AddressEventSource, type Address } from "@/lib/api";
import { useDeviceAddresses } from "@/features/devices/hooks/useDeviceAddresses";
import { useAddDeviceAddress } from "@/features/devices/hooks/useAddDeviceAddress";
import { useDisableDeviceAddress } from "@/features/devices/hooks/useDisableDeviceAddress";
import { useDeviceHeartbeat } from "@/features/devices/hooks/useDeviceHeartbeat";
import classes from "./DeviceAddressesTab.module.css";

dayjs.extend(relativeTime);

const STALE_THRESHOLD_DAYS = 7;

function isStale(address: Address): boolean {
  return !address.is_enabled && dayjs().diff(dayjs(address.updated_at), "day") > STALE_THRESHOLD_DAYS;
}

function isActive(address: Address): boolean {
  return !isStale(address);
}

const SOURCE_LABELS: Record<string, string> = {
  [AddressEventSource.HEARTBEAT]: "heartbeat",
  [AddressEventSource.MANUAL]: "manual",
  [AddressEventSource.EXPIRY]: "expired",
  [AddressEventSource.LIMIT_EXCEEDED]: "evicted",
};

function formatDuration(fromIso: string, toIso: string): string {
  const mins = dayjs(toIso).diff(dayjs(fromIso), "minute");
  if (mins < 60) return `${mins}m`;
  const h = Math.floor(mins / 60);
  const m = mins % 60;
  return m > 0 ? `${h}h ${m}m` : `${h}h`;
}

function TtlBar({ address }: { address: Address }) {
  if (!address.is_enabled) {
    if (!address.updated_at) return null;
    return (
      <Text size="xs" c="dimmed">
        inactive {dayjs(address.updated_at).fromNow(true)} ago
      </Text>
    );
  }
  if (!address.expires_at) return null;

  const now = dayjs();
  const expiresAt = dayjs(address.expires_at);
  const updatedAt = dayjs(address.updated_at);
  const total = expiresAt.diff(updatedAt, "second");
  const remaining = expiresAt.diff(now, "second");
  const pct = total > 0 ? Math.max(0, Math.min(100, (remaining / total) * 100)) : 0;
  const color = pct < 15 ? "red" : pct < 40 ? "orange" : "indigo";

  return (
    <Box style={{ minWidth: 60 }}>
      <Progress value={pct} size="xs" color={color} style={{ marginBottom: 2 }} />
      <Text size="xs" c="dimmed">
        {remaining > 0 ? `${Math.round(remaining / 60)}m left` : "expired"}
      </Text>
    </Box>
  );
}

function StateDot({ enabled }: { enabled: boolean }) {
  return (
    <Tooltip label={enabled ? "live" : "inactive"} withArrow>
      <Box
        component="span"
        style={{
          display: "inline-block",
          width: 8,
          height: 8,
          borderRadius: "50%",
          flexShrink: 0,
          background: enabled
            ? "var(--mantine-color-orange-4)"
            : "var(--mantine-color-dimmed)",
        }}
      />
    </Tooltip>
  );
}

function AddressRow({
  address,
  formatDateTime,
  onToggle,
  togglePending,
}: {
  address: Address;
  formatDateTime: (iso: string) => string;
  onToggle: (a: Address) => void;
  togglePending: boolean;
}) {
  const prevEnabled = useRef(address.is_enabled);
  const [highlight, setHighlight] = useState(false);

  useEffect(() => {
    if (!prevEnabled.current && address.is_enabled) {
      setHighlight(true);
    }
    prevEnabled.current = address.is_enabled;
  }, [address.is_enabled]);

  return (
    <Table.Tr
      key={address.id}
      className={highlight ? classes.rowHighlight : undefined}
      onAnimationEnd={() => setHighlight(false)}
    >
      <Table.Td ff="monospace" fz="sm">{address.ip}</Table.Td>
      <Table.Td>
        <Group gap={6} wrap="nowrap">
          <StateDot enabled={address.is_enabled} />
          <Text size="xs" c="dimmed">{address.is_enabled ? "live" : "inactive"}</Text>
        </Group>
      </Table.Td>
      <Table.Td>
        <Text size="xs" c="dimmed">
          {formatDateTime(address.updated_at)} · {SOURCE_LABELS[address.source] ?? address.source}
        </Text>
      </Table.Td>
      <Table.Td>
        <Text size="xs" c="dimmed">
          {formatDuration(address.created_at, dayjs().toISOString())}
        </Text>
      </Table.Td>
      <Table.Td>
        <TtlBar address={address} />
      </Table.Td>
      <Table.Td>
        <Switch
          size="xs"
          checked={address.is_enabled}
          onChange={() => onToggle(address)}
          disabled={togglePending}
        />
      </Table.Td>
    </Table.Tr>
  );
}

interface DeviceAddressesTabProps {
  deviceId: number;
  isDisabled?: boolean;
}

export function DeviceAddressesTab({ deviceId, isDisabled = false }: DeviceAddressesTabProps) {
  const formatDateTime = useDateFormatter();
  const { data: addresses, isLoading, isError, error, refetch } = useDeviceAddresses(deviceId, true, 10_000);

  const addMutation = useAddDeviceAddress();
  const disableMutation = useDisableDeviceAddress();
  const heartbeatMutation = useDeviceHeartbeat();

  const [view, setView] = useState<"active" | "stale">("active");
  const [customOpen, setCustomOpen] = useState(false);
  const [customIp, setCustomIp] = useState("");
  const [addressSearch, setAddressSearch] = useState("");
  const [proxyConflict, setProxyConflict] = useState(false);

  const activeAddresses = useMemo(
    () =>
      (addresses ?? [])
        .filter(isActive)
        .sort((a, b) => Number(b.is_enabled) - Number(a.is_enabled)),
    [addresses],
  );
  const staleAddresses = useMemo(
    () => (addresses ?? []).filter(isStale),
    [addresses],
  );

  const filteredActive = useMemo(() => {
    if (!addressSearch.trim()) return activeAddresses;
    const q = addressSearch.toLowerCase();
    return activeAddresses.filter((a) => a.ip.toLowerCase().includes(q));
  }, [activeAddresses, addressSearch]);

  const filteredStale = useMemo(() => {
    if (!addressSearch.trim()) return staleAddresses;
    const q = addressSearch.toLowerCase();
    return staleAddresses.filter((a) => a.ip.toLowerCase().includes(q));
  }, [staleAddresses, addressSearch]);

  function handleHeartbeat() {
    heartbeatMutation.mutate(
      { path: { device_id: deviceId } },
      {
        onSuccess: (address) => {
          setProxyConflict(false);
          notifications.show({ color: "green", message: `IP ${address.ip} registered` });
        },
        onError: (err) => {
          const message = toErrorMessage(err);
          if (message.includes("Trusted proxy")) {
            setProxyConflict(true);
          } else {
            notifications.show({ color: "red", message });
          }
        },
      },
    );
  }

  function handleCustomSubmit() {
    if (!customIp.trim()) return;
    addMutation.mutate(
      { path: { device_id: deviceId }, body: { ip: customIp.trim() } },
      {
        onSuccess: () => {
          setProxyConflict(false);
          notifications.show({ color: "green", message: "Address added" });
          setCustomIp("");
          setCustomOpen(false);
        },
        onError: (err) => {
          const message = toErrorMessage(err);
          if (message.includes("Trusted proxy")) {
            setProxyConflict(true);
          } else {
            notifications.show({ color: "red", title: "Error adding address", message });
          }
        },
      },
    );
  }

  function handleToggle(address: Address) {
    if (address.is_enabled) {
      disableMutation.mutate(
        { path: { device_id: deviceId, address_id: address.id } },
        {
          onError: (err) =>
            notifications.show({ color: "red", message: toErrorMessage(err) }),
        },
      );
    } else {
      addMutation.mutate(
        { path: { device_id: deviceId }, body: { ip: address.ip } },
        {
          onError: (err) =>
            notifications.show({ color: "red", message: toErrorMessage(err) }),
        },
      );
    }
  }

  function handleReEnable(address: Address) {
    addMutation.mutate(
      { path: { device_id: deviceId }, body: { ip: address.ip } },
      {
        onSuccess: () =>
          notifications.show({ color: "green", message: "Address re-enabled" }),
        onError: (err) =>
          notifications.show({ color: "red", message: toErrorMessage(err) }),
      },
    );
  }

  const togglePending = disableMutation.isPending || addMutation.isPending;

  return (
    <Stack gap="md">
      {/* Register actions */}
      <Group gap="xs">
        <Button
          size="xs"
          onClick={handleHeartbeat}
          loading={heartbeatMutation.isPending}
          disabled={isDisabled}
        >
          ＋ Register my IP
        </Button>
        <Button
          size="xs"
          variant="subtle"
          onClick={() => setCustomOpen((o) => !o)}
          disabled={isDisabled}
        >
          ＋ Custom IP…
        </Button>
      </Group>

      {proxyConflict && (
        <Alert
          icon={<IconAlertTriangle size={16} />}
          title="Reverse proxy IP detected"
          color="orange"
          withCloseButton
          onClose={() => setProxyConflict(false)}
        >
          <Text size="xs">
            PulseWeaver received your reverse proxy's IP instead of yours. This usually means{" "}
            <Code>X-Real-IP</Code> is not being forwarded to PulseWeaver by your proxy.{" "}
            <Anchor
              href="https://github.com/diegoguidaf/pulseweaver/blob/main/docs/Caddy-Setup.md#trusted-proxy-ip-addresses-cannot-be-registered"
              target="_blank"
              rel="noopener noreferrer"
              size="xs"
            >
              How to fix your proxy configuration →
            </Anchor>
          </Text>
        </Alert>
      )}

      <Collapse expanded={customOpen}>
        <Group gap="xs" align="flex-end">
          <TextInput
            size="xs"
            placeholder="192.168.1.100 or 2001:db8::1"
            value={customIp}
            onChange={(e) => setCustomIp(e.currentTarget.value)}
            onKeyDown={(e) => e.key === "Enter" && handleCustomSubmit()}
            style={{ width: 200 }}
            autoFocus
          />
          <Button size="xs" onClick={handleCustomSubmit} loading={addMutation.isPending}>
            Add
          </Button>
          <ActionIcon
            size="xs"
            variant="subtle"
            color="gray"
            onClick={() => { setCustomOpen(false); setCustomIp(""); }}
          >
            <IconX size={12} />
          </ActionIcon>
        </Group>
      </Collapse>

      {isError ? (
        <ErrorState error={error} title="Failed to load addresses" onRetry={() => refetch()} />
      ) : (
        <>
      {/* Active / Stale toggle */}
      {isLoading ? (
        <Skeleton height={32} width={180} />
      ) : (
        <Stack gap={6}>
          <SegmentedControl
            size="xs"
            value={view}
            onChange={(v) => setView(v as "active" | "stale")}
            data={[
              { label: `Active · ${activeAddresses.length}`, value: "active" },
              { label: `Stale · ${staleAddresses.length}`, value: "stale" },
            ]}
          />
          {view === "stale" && (
            <Text size="xs" c="dimmed">
              Addresses with no activity for more than {STALE_THRESHOLD_DAYS} days — re-enable to make them live again.
            </Text>
          )}
        </Stack>
      )}

      {/* Shared IP filter */}
      <TextInput
        size="xs"
        placeholder="Filter by IP…"
        leftSection={<IconSearch size={12} />}
        value={addressSearch}
        onChange={(e) => setAddressSearch(e.currentTarget.value)}
        style={{ maxWidth: 240 }}
      />

      {/* Address table */}
      {isLoading ? (
        <Stack gap={8}>
          <Skeleton height={16} />
          <Skeleton height={16} />
          <Skeleton height={16} width="60%" />
        </Stack>
      ) : view === "active" ? (
        filteredActive.length === 0 ? (
          <Text size="sm" c="dimmed">
            {addressSearch ? "No addresses match." : "No active addresses."}
          </Text>
        ) : (
          <Table highlightOnHover>
            <Table.Thead>
              <Table.Tr>
                <Table.Th>IP</Table.Th>
                <Table.Th>State</Table.Th>
                <Table.Th>Updated · via</Table.Th>
                <Table.Th>Lifetime</Table.Th>
                <Table.Th>Expires in</Table.Th>
                <Table.Th w={48} />
              </Table.Tr>
            </Table.Thead>
            <Table.Tbody>
              {filteredActive.map((a) => (
                <AddressRow
                  key={a.id}
                  address={a}
                  formatDateTime={formatDateTime}
                  onToggle={handleToggle}
                  togglePending={togglePending}
                />
              ))}
            </Table.Tbody>
          </Table>
        )
      ) : (
        filteredStale.length === 0 ? (
          <Text size="sm" c="dimmed">
            {addressSearch ? "No addresses match." : "No stale addresses."}
          </Text>
        ) : (
          <Table highlightOnHover>
            <Table.Thead>
              <Table.Tr>
                <Table.Th>IP</Table.Th>
                <Table.Th>Last seen · via</Table.Th>
                <Table.Th>Inactive since</Table.Th>
                <Table.Th>Lifetime</Table.Th>
                <Table.Th w={90} />
              </Table.Tr>
            </Table.Thead>
            <Table.Tbody>
              {filteredStale.map((a) => (
                <Table.Tr key={a.id}>
                  <Table.Td ff="monospace" fz="sm">{a.ip}</Table.Td>
                  <Table.Td>
                    <Text size="xs" c="dimmed">
                      {formatDateTime(a.updated_at)} · {SOURCE_LABELS[a.source] ?? a.source}
                    </Text>
                  </Table.Td>
                  <Table.Td>
                    <Text size="xs" c="dimmed">{formatDateTime(a.updated_at)}</Text>
                  </Table.Td>
                  <Table.Td>
                    <Text size="xs" c="dimmed">
                      {formatDuration(a.created_at, a.updated_at)}
                    </Text>
                  </Table.Td>
                  <Table.Td>
                    <Button
                      size="xs"
                      variant="subtle"
                      onClick={() => handleReEnable(a)}
                      loading={addMutation.isPending}
                    >
                      Re-enable
                    </Button>
                  </Table.Td>
                </Table.Tr>
              ))}
            </Table.Tbody>
          </Table>
        )
      )}
        </>
      )}
    </Stack>
  );
}
