import React, { useEffect, useMemo } from "react";
import { Link, Navigate, useNavigate, useParams, useSearchParams } from "react-router-dom";
import { ROUTES, buildRoute } from "@/lib/routes";
import dayjs from "dayjs";
import relativeTime from "dayjs/plugin/relativeTime";
import {
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
import { useLocalStorage, useMediaQuery } from "@mantine/hooks";
import { IconChevronLeft } from "@tabler/icons-react";
import classes from "./UserDevicesPage.module.css";
import { ErrorState } from "@/components/ErrorState";
import { resolveDeviceIcon } from "@/features/devices/deviceTypeConfig";
import { RuleChips } from "@/features/devices/RuleChips";
import { useOwnerGroup } from "@/features/devices/hooks/useOwnerGroup";
import { OwnerDevicesPanel } from "@/features/devices/OwnerDevicesPanel";
import { DeviceAddressesTab } from "@/features/devices/DeviceAddressesTab";
import { DeviceRulesTab } from "@/features/devices/DeviceRulesTab";
import { DeviceHistoryTab } from "@/features/devices/DeviceHistoryTab";
import { DeviceSettingsTab, type DeviceData } from "@/features/devices/DeviceSettingsTab";
import {
  DeviceCreatePane,
  DeviceCreateEmptyState,
} from "@/features/devices/DeviceCreatePane";
import { DeviceStatusBadge } from "@/features/devices/DeviceStatusBadge";
import { DevicePairingBanner } from "@/features/device-pairing/DevicePairingBanner";
import { DevicePairingTab } from "@/features/device-pairing/DevicePairingTab";
import { DeviceState } from "@/lib/api";
import type { DeviceRuleSummary } from "@/lib/api";
import { DeviceApiKeyRuleHintBanner } from "@/features/devices/DeviceApiKeyRuleHintBanner";
import { DeviceDisabledBanner } from "@/features/devices/DeviceDisabledBanner";

dayjs.extend(relativeTime);


function formatCreatedAt(iso: string): string {
  return dayjs(iso).format("D MMM YYYY");
}

function hasActiveLimitRule(rules: Array<DeviceRuleSummary>): boolean {
  return rules.some((r) => r.enabled);
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

interface UserDevicesPageProps {
  /** Rendered at /devices/owners/:id/new — shows the in-pane create form. */
  createMode?: boolean;
}

export function UserDevicesPage({ createMode = false }: UserDevicesPageProps) {
  const { ownerId: ownerIdParam } = useParams<RouteParams>();
  const navigate = useNavigate();
  const [searchParams, setSearchParams] = useSearchParams();
  const [sidebarWidth, setSidebarWidth] = useLocalStorage({
    key: "pw-device-sidebar-width",
    defaultValue: 280,
    getInitialValueInEffect: false,
  });
  const isDesktop = useMediaQuery("(min-width: 62em)", true, { getInitialValueInEffect: false });

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

  const ownerId = ownerIdParam ? Number.parseInt(ownerIdParam, 10) : Number.NaN;
  const deviceIdStr = searchParams.get("device");
  const deviceId = deviceIdStr ? Number.parseInt(deviceIdStr, 10) : undefined;

  const { data: group, isLoading, error, refetch } = useOwnerGroup(ownerId);

  const selectedDevice = useMemo(
    () => (deviceId !== undefined ? group?.devices.find((d) => d.id === deviceId) : undefined),
    [deviceId, group],
  );

  // description is absent from DeviceListEntry; defaulted to null until the list
  // entry carries it. The profile card sends description on save only when changed,
  // so the default cannot silently overwrite the real value.
  const deviceData = useMemo<DeviceData | undefined>(() => {
    if (!selectedDevice || !group) return undefined;
    return {
      name: selectedDevice.name,
      api_key_prefix: selectedDevice.api_key_prefix ?? null,
      description: null,
      icon: selectedDevice.icon ?? null,
      state: selectedDevice.state,
      owner_id: group.owner.id,
      owner_name: group.owner.display_name,
      created_at: selectedDevice.created_at ?? null,
    };
  }, [selectedDevice, group]);

  // Auto-select first device when no ?device= param is present (not while creating)
  useEffect(() => {
    if (!createMode && deviceId === undefined && group?.devices.length) {
      setSearchParams({ device: String(group.devices[0].id) }, { replace: true });
    }
  }, [createMode, deviceId, group, setSearchParams]);

  const renderDeviceIcon = resolveDeviceIcon(selectedDevice?.icon);

  function goToDevice(id: number, tab: "addresses" | "pairing") {
    navigate(`${buildRoute.userDevices(ownerId)}?device=${id}&tab=${tab}`);
  }

  const hasNoDevices = Boolean(group) && group?.devices.length === 0;

  if (!ownerIdParam || Number.isNaN(ownerId)) {
    return <Navigate to={ROUTES.devices} replace />;
  }

  return (
    <Group
      align="stretch"
      gap={0}
      wrap={isDesktop ? "nowrap" : "wrap"}
      style={{ maxWidth: 1280, width: "100%" }}
    >
      {/* Left sidebar — full-width row above the content below the AppShell's md breakpoint;
          the drag-to-resize handle only makes sense with a mouse, so it's desktop-only too. */}
      <Box
        pr={isDesktop ? "lg" : 0}
        pb={isDesktop ? 0 : "lg"}
        style={{
          width: isDesktop ? sidebarWidth : "100%",
          flexShrink: 0,
          position: "relative",
          borderRight: isDesktop ? "1px solid var(--mantine-color-default-border)" : "none",
          borderBottom: isDesktop ? "none" : "1px solid var(--mantine-color-default-border)",
        }}
      >
        {isDesktop && <Box className={classes.resizeHandle} onMouseDown={handleResizeMouseDown} />}
        <Stack gap="lg">
          <Anchor
            component={Link}
            to={ROUTES.devices}
            c="dimmed"
            size="sm"
            style={{ display: "inline-flex", alignItems: "center", gap: 4, minHeight: 24 }}
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
            <ErrorState error={error} title="Could not load devices" onRetry={() => refetch()} />
          ) : group ? (
            <OwnerDevicesPanel
              owner={group.owner}
              devices={group.devices}
              selectedDeviceId={deviceId}
              onSelectDevice={(id) =>
                setSearchParams((prev) => {
                  prev.set("device", String(id));
                  return prev;
                })
              }
              onAddDevice={() => navigate(buildRoute.userDevicesNew(ownerId))}
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
      <Stack
        pl={isDesktop ? "xl" : 0}
        pt={isDesktop ? 0 : "lg"}
        gap="lg"
        style={{ flex: 1, minWidth: isDesktop ? 0 : "100%" }}
      >
        {createMode && group ? (
          <DeviceCreatePane
            ownerId={ownerId}
            ownerName={group.owner.display_name}
            onCancel={() => navigate(buildRoute.userDevices(ownerId))}
            onCreated={goToDevice}
          />
        ) : hasNoDevices && group ? (
          <DeviceCreateEmptyState
            ownerName={group.owner.display_name}
            onCreate={() => navigate(buildRoute.userDevicesNew(ownerId))}
          />
        ) : (
          <>
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
              <Title order={1} size="h3">{selectedDevice.name}</Title>
              <DeviceStatusBadge state={selectedDevice.state} size="sm" />
              <RuleChips entry={selectedDevice} size="xs" />
            </Group>
            <Group gap={6} wrap="wrap">
              {selectedDevice.live_address_count > 0 && (
                <Text size="xs" c="var(--pw-amber-text)">
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

        {/* Disabled banner */}
        {selectedDevice?.state === DeviceState.DISABLED && (
          <DeviceDisabledBanner deviceId={selectedDevice.id} />
        )}

        {/* API key + no-limits hint banner */}
        {selectedDevice?.api_key_prefix && !hasActiveLimitRule(selectedDevice.rules) && (
          <DeviceApiKeyRuleHintBanner
            deviceId={selectedDevice.id}
            onGoToRules={() =>
              setSearchParams((prev) => {
                prev.set("tab", DeviceTab.RULES);
                return prev;
              })
            }
            onGoToSettings={() =>
              setSearchParams((prev) => {
                prev.set("tab", DeviceTab.SETTINGS);
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
              <DeviceAddressesTab
                deviceId={selectedDevice.id}
                isDisabled={selectedDevice.state === DeviceState.DISABLED}
              />
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
              <DeviceSettingsTab
                deviceId={selectedDevice.id}
                device={deviceData}
                onDeviceDeleted={() =>
                  setSearchParams((prev) => {
                    prev.delete("device");
                    prev.delete("tab");
                    return prev;
                  })
                }
              />
            </Tabs.Panel>
          </Tabs>
        )}
          </>
        )}
      </Stack>
    </Group>
  );
}
