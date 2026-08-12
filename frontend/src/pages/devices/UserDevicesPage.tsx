import React, { useEffect, useMemo } from "react";
import { Link, Navigate, useNavigate, useParams, useSearchParams } from "react-router";
import { ROUTES, buildRoute, DeviceTab, type DeviceTabValue } from "@/lib/routes";
import { Anchor, Box, Button, Group, Indicator, Skeleton, Stack, Tabs, Text } from "@mantine/core";
import { useLocalStorage, useMediaQuery } from "@mantine/hooks";
import { IconChevronLeft, IconUserQuestion } from "@tabler/icons-react";
import classes from "./UserDevicesPage.module.css";
import { EmptyState } from "@/components/EmptyState";
import { ErrorState } from "@/components/ErrorState";
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
import { DevicePairingTab } from "@/features/device-pairing/DevicePairingTab";
import { DevicePairingStatus, DeviceState } from "@/lib/api";
import { DeviceWorkspaceBanners } from "@/features/devices/DeviceWorkspaceBanners";
import { DeviceWorkspaceHeader } from "@/features/devices/DeviceWorkspaceHeader";

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

  const { data: group, isPending, error, refetch } = useOwnerGroup(ownerId);

  const selectedDevice = useMemo(
    () => (deviceId !== undefined ? group?.devices.find((d) => d.id === deviceId) : undefined),
    [deviceId, group],
  );

  const deviceData = useMemo<DeviceData | undefined>(() => {
    if (!selectedDevice || !group) return undefined;
    return {
      name: selectedDevice.name,
      api_key_prefix: selectedDevice.api_key_prefix ?? null,
      description: selectedDevice.description ?? null,
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

          {isPending ? (
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
        {!isPending && !error && !group ? (
          <EmptyState
            icon={IconUserQuestion}
            title="User not found"
            description="This owner does not exist, or has been deleted."
            action={
              <Button component={Link} to={ROUTES.devices} variant="light">
                Back to devices
              </Button>
            }
          />
        ) : createMode && group ? (
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
        {isPending && !selectedDevice ? (
          <Stack gap={6}>
            <Skeleton height={22} width={200} />
            <Skeleton height={14} width={280} />
          </Stack>
        ) : selectedDevice ? (
          <DeviceWorkspaceHeader device={selectedDevice} />
        ) : null}

        {selectedDevice && (
          <DeviceWorkspaceBanners
            device={selectedDevice}
            ownerId={ownerId}
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
                    selectedDevice.pairing?.status !== DevicePairingStatus.PENDING &&
                    selectedDevice.pairing?.status !== DevicePairingStatus.EXPIRED
                  }
                  color={
                    selectedDevice.pairing?.status === DevicePairingStatus.EXPIRED
                      ? "red"
                      : "indigo"
                  }
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
                ownerId={ownerId}
                isDisabled={selectedDevice.state === DeviceState.DISABLED}
              />
            </Tabs.Panel>
            <Tabs.Panel value={DeviceTab.RULES} pt="md">
              <DeviceRulesTab deviceId={selectedDevice.id} ownerId={ownerId} />
            </Tabs.Panel>
            <Tabs.Panel value={DeviceTab.PAIRING} pt="md">
              <DevicePairingTab
                deviceId={selectedDevice.id}
                ownerId={ownerId}
                deviceState={selectedDevice.state}
              />
            </Tabs.Panel>
            <Tabs.Panel value={DeviceTab.HISTORY} pt="md">
              <DeviceHistoryTab deviceId={selectedDevice.id} />
            </Tabs.Panel>
            <Tabs.Panel value={DeviceTab.SETTINGS} pt="md">
              <DeviceSettingsTab
                deviceId={selectedDevice.id}
                ownerId={ownerId}
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
