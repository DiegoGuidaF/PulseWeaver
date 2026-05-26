import React, { useEffect, useMemo, useState } from "react";
import { Link, Navigate, useParams, useSearchParams } from "react-router-dom";
import { ROUTES } from "@/lib/routes";
import dayjs from "dayjs";
import relativeTime from "dayjs/plugin/relativeTime";
import {
  Alert,
  Anchor,
  Box,
  Group,
  Indicator,
  Skeleton,
  Stack,
  Tabs,
  Text,
  ThemeIcon,
  Title,
} from "@mantine/core";
import { useLocalStorage } from "@mantine/hooks";
import { IconAlertCircle, IconChevronLeft } from "@tabler/icons-react";
import classes from "./UserDevicesPage.module.css";
import { toErrorMessage } from "@/lib/api-client";
import { resolveDeviceIcon } from "@/features/devices/deviceTypeConfig";
import { RuleChips } from "@/features/devices/RuleChips";
import { useOwnerGroup } from "@/features/devices/hooks/useOwnerGroup";
import { OwnerDevicesPanel } from "@/features/devices/OwnerDevicesPanel";
import { DeviceAddressesTab } from "@/features/devices/DeviceAddressesTab";
import { DeviceRulesTab } from "@/features/devices/DeviceRulesTab";
import { DeviceHistoryTab } from "@/features/devices/DeviceHistoryTab";
import { DeviceSettingsTab, type DeviceData } from "@/features/devices/DeviceSettingsTab";
import { CreateDeviceModal } from "@/features/devices/CreateDeviceModal";
import { DevicePairingBanner } from "@/features/device-pairing/DevicePairingBanner";
import { DevicePairingTab } from "@/features/device-pairing/DevicePairingTab";
import { DeviceState } from "@/lib/api";

dayjs.extend(relativeTime);


function formatCreatedAt(iso: string): string {
  return dayjs(iso).format("D MMM YYYY");
}

const DeviceTab = {
  ADDRESSES: "addresses",
  RULES: "rules",
  PAIRING: "pairing",
  HISTORY: "history",
  SETTINGS: "settings",
} as const;

type DeviceTabValue = (typeof DeviceTab)[keyof typeof DeviceTab];
const VALID_DEVICE_TABS = new Set<string>(Object.values(DeviceTab));
function resolveTab(raw: string | null): DeviceTabValue {
  return raw !== null && VALID_DEVICE_TABS.has(raw) ? (raw as DeviceTabValue) : DeviceTab.ADDRESSES;
}

type RouteParams = { ownerId?: string };

export function UserDevicesPage() {
  const { ownerId: ownerIdParam } = useParams<RouteParams>();
  const [searchParams, setSearchParams] = useSearchParams();
  const [sidebarWidth, setSidebarWidth] = useLocalStorage({
    key: "pw-device-sidebar-width",
    defaultValue: 280,
    getInitialValueInEffect: false,
  });

  function handleResizeMouseDown(e: React.MouseEvent) {
    e.preventDefault();
    const startX = e.clientX;
    const startWidth = sidebarWidth;

    const onMove = (ev: MouseEvent) => {
      setSidebarWidth(Math.max(180, Math.min(450, startWidth + (ev.clientX - startX))));
    };
    const onUp = () => {
      document.removeEventListener("mousemove", onMove);
      document.removeEventListener("mouseup", onUp);
    };
    document.addEventListener("mousemove", onMove);
    document.addEventListener("mouseup", onUp);
  }

  const [createOpen, setCreateOpen] = useState(false);

  const ownerId = ownerIdParam ? Number.parseInt(ownerIdParam, 10) : Number.NaN;
  const deviceIdStr = searchParams.get("device");
  const deviceId = deviceIdStr ? Number.parseInt(deviceIdStr, 10) : undefined;

  const { data: group, isLoading, error } = useOwnerGroup(ownerId);

  const selectedDevice = useMemo(
    () => (deviceId !== undefined ? group?.devices.find((d) => d.id === deviceId) : undefined),
    [deviceId, group],
  );

  // TODO: device_type and description are absent from DeviceListEntry; using safe defaults
  // until the API spec is extended. The profile card won't send device_type on save unless
  // the user explicitly changes it, so the default cannot silently overwrite the real value.
  const deviceData = useMemo<DeviceData | undefined>(() => {
    if (!selectedDevice || !group) return undefined;
    return {
      name: selectedDevice.name,
      api_key_prefix: selectedDevice.api_key_prefix ?? null,
      device_type: "static",
      description: null,
      icon: selectedDevice.icon ?? null,
      owner_id: group.owner.id,
      owner_name: group.owner.display_name,
    };
  }, [selectedDevice, group]);

  // Auto-select first device when no ?device= param is present
  useEffect(() => {
    if (deviceId === undefined && group?.devices.length) {
      setSearchParams({ device: String(group.devices[0].id) }, { replace: true });
    }
  }, [deviceId, group, setSearchParams]);

  const renderDeviceIcon = resolveDeviceIcon(selectedDevice?.icon);

  if (!ownerIdParam || Number.isNaN(ownerId)) {
    return <Navigate to={ROUTES.devices} replace />;
  }

  return (
    <>
    <CreateDeviceModal
      opened={createOpen}
      onClose={() => setCreateOpen(false)}
      defaultOwnerId={group?.owner.id ?? null}
    />
    <Group
      align="stretch"
      gap={0}
      wrap="nowrap"
      style={{ maxWidth: 1280, width: "100%" }}
    >
      {/* Left sidebar */}
      <Box
        pr="lg"
        style={{
          width: sidebarWidth,
          flexShrink: 0,
          position: "relative",
          borderRight: "1px solid var(--mantine-color-default-border)",
        }}
      >
        <Box className={classes.resizeHandle} onMouseDown={handleResizeMouseDown} />
        <Stack gap="lg">
          <Anchor
            component={Link}
            to={ROUTES.devices}
            c="dimmed"
            size="sm"
            style={{ display: "inline-flex", alignItems: "center", gap: 4 }}
          >
            <IconChevronLeft size={16} stroke={1.5} />
            <span>Devices · all owners</span>
          </Anchor>

          {isLoading ? (
            <Stack gap="sm">
              <Group gap="sm">
                <Skeleton circle height={40} />
                <Stack gap={4}>
                  <Skeleton height={14} width={120} />
                  <Skeleton height={12} width={80} />
                </Stack>
              </Group>
              <Skeleton height={12} width={100} mt="xs" />
              <Skeleton height={36} radius="sm" />
              <Skeleton height={36} radius="sm" />
              <Skeleton height={36} radius="sm" />
            </Stack>
          ) : error ? (
            <Alert color="red" icon={<IconAlertCircle size={16} />} title="Could not load devices">
              {toErrorMessage(error)}
            </Alert>
          ) : group ? (
            <OwnerDevicesPanel
              owner={group.owner}
              devices={group.devices}
              selectedDeviceId={deviceId}
              onSelectDevice={(id) => setSearchParams({ device: String(id) })}
              onAddDevice={() => setCreateOpen(true)}
            />
          ) : (
            <Text size="sm" c="dimmed">
              User not found.{" "}
              <Anchor component={Link} to={ROUTES.devices}>Back to devices</Anchor>
            </Text>
          )}
        </Stack>
      </Box>

      {/* Right main content */}
      <Stack pl="xl" gap="lg" style={{ flex: 1, minWidth: 0 }}>
        {/* Device header */}
        {isLoading && !selectedDevice ? (
          <Stack gap={6}>
            <Skeleton height={22} width={200} />
            <Skeleton height={14} width={280} />
          </Stack>
        ) : selectedDevice ? (
          <Stack gap={4}>
            <Group gap="xs" align="center">
              <ThemeIcon variant="transparent" size="md" c="dimmed">
                {renderDeviceIcon({ size: 22 })}
              </ThemeIcon>
              <Title order={3}>{selectedDevice.name}</Title>
              <RuleChips entry={selectedDevice} size="xs" />
            </Group>
            <Group gap={6} wrap="wrap">
              {selectedDevice.live_address_count > 0 && (
                <Text size="xs" c="orange.4">
                  live · {selectedDevice.live_address_count} IP{selectedDevice.live_address_count !== 1 ? "s" : ""}
                </Text>
              )}
              {selectedDevice.last_seen_at && (
                <>
                  <Text size="xs" c="dimmed">·</Text>
                  <Text size="xs" c="dimmed">seen {dayjs(selectedDevice.last_seen_at).fromNow()}</Text>
                </>
              )}
              {selectedDevice.api_key_prefix && (
                <>
                  <Text size="xs" c="dimmed">·</Text>
                  <Text size="xs" c="dimmed" ff="monospace">{selectedDevice.api_key_prefix}…</Text>
                </>
              )}
              {selectedDevice.created_at && (
                <>
                  <Text size="xs" c="dimmed">·</Text>
                  <Text size="xs" c="dimmed">created {formatCreatedAt(selectedDevice.created_at)}</Text>
                </>
              )}
            </Group>
          </Stack>
        ) : null}

        {/* Pairing banner — shown when a code is outstanding */}
        {selectedDevice?.state === DeviceState.PENDING_CLAIM && selectedDevice.pairing && (
          <DevicePairingBanner
            expiresAt={selectedDevice.pairing.expires_at}
            onViewPairing={() =>
              setSearchParams((prev) => {
                prev.set("tab", DeviceTab.PAIRING);
                return prev;
              })
            }
          />
        )}

        {/* Tabs — only rendered when a valid device is selected */}
        {selectedDevice && (
          <Tabs
            key={selectedDevice.id}
            value={resolveTab(searchParams.get("tab"))}
            onChange={(value) =>
              setSearchParams((prev) => {
                prev.set("tab", resolveTab(value));
                return prev;
              })
            }
            keepMounted={false}
          >
            <Tabs.List>
              <Tabs.Tab value={DeviceTab.ADDRESSES}>Addresses</Tabs.Tab>
              <Tabs.Tab value={DeviceTab.RULES}>Rules</Tabs.Tab>
              <Tabs.Tab value={DeviceTab.PAIRING}>
                <Indicator
                  disabled={
                    selectedDevice.state !== DeviceState.PENDING_CLAIM &&
                    selectedDevice.state !== DeviceState.EXPIRED_CLAIM
                  }
                  color={selectedDevice.state === DeviceState.EXPIRED_CLAIM ? "red" : "indigo"}
                  size={6}
                  offset={-2}
                >
                  Pairing
                </Indicator>
              </Tabs.Tab>
              <Tabs.Tab value={DeviceTab.HISTORY}>History</Tabs.Tab>
              <Tabs.Tab value={DeviceTab.SETTINGS}>Settings</Tabs.Tab>
            </Tabs.List>
            <Tabs.Panel value={DeviceTab.ADDRESSES} pt="md">
              <DeviceAddressesTab deviceId={selectedDevice.id} />
            </Tabs.Panel>
            <Tabs.Panel value={DeviceTab.RULES} pt="md">
              <DeviceRulesTab deviceId={selectedDevice.id} liveAddressCount={selectedDevice.live_address_count} />
            </Tabs.Panel>
            <Tabs.Panel value={DeviceTab.PAIRING} pt="md">
              <DevicePairingTab deviceId={selectedDevice.id} deviceState={selectedDevice.state} />
            </Tabs.Panel>
            <Tabs.Panel value={DeviceTab.HISTORY} pt="md">
              <DeviceHistoryTab deviceId={selectedDevice.id} />
            </Tabs.Panel>
            <Tabs.Panel value={DeviceTab.SETTINGS} pt="md">
              <DeviceSettingsTab deviceId={selectedDevice.id} device={deviceData} />
            </Tabs.Panel>
          </Tabs>
        )}
      </Stack>
    </Group>
    </>
  );
}
