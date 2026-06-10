import {
    Anchor,
    Badge,
    Group,
    SegmentedControl,
    Select,
    Stack,
    Text,
    TextInput,
    ThemeIcon,
    Tooltip,
} from "@mantine/core";
import { DateTimePicker } from "@mantine/dates";
import { IconArrowsRightLeft, IconSearch } from "@tabler/icons-react";
import type { DataTableColumn } from "mantine-datatable";
import type { AddressHistoryEvent } from "@/lib/api";
import type { AddressHistoryFilters } from "../hooks/useAddressHistoryFilters";
import { SOURCE_LABELS, formatGapDuration } from "../constants";
import dayjs from "dayjs";

const refDateStyle = {
    outline: "1.5px dashed var(--mantine-color-dimmed)",
    outlineOffset: -1.5,
    borderRadius: "var(--mantine-radius-sm)",
} as const;

function renderGapCell({ gapSeconds, ttlSeconds }: { gapSeconds: number | null | undefined; ttlSeconds: number | null | undefined }) {
    if (gapSeconds == null) {
        return <Text size="sm" ff="monospace" c="dimmed">—</Text>;
    }

    const ratio = ttlSeconds ? gapSeconds / ttlSeconds : null;
    let color: string | undefined;
    let status: string | null = null;
  if (ratio != null && ratio >= 1) {
    color = "red";
    status = "Above the TTL — Device lost access, consider raising the auto-expiry TTL or heartbeat frequency";
  }else if (ratio != null && ratio > 0.9) {
        color = "red";
        status = "Too close to the TTL — consider raising the auto-expiry TTL or heartbeat frequency";
    } else if (ratio != null && ratio > 0.7) {
        color = "yellow";
        status = "Approaching the TTL — keep an eye on heartbeat regularity";
    }

    const label = (
        <Text size="sm" ff="monospace" c={color ?? "dimmed"}>
            {formatGapDuration(gapSeconds)}
        </Text>
    );

    if (ttlSeconds == null) return label;

    return (
        <Tooltip
            label={
                <Stack gap={2}>
                    <Text size="xs">Device TTL: {formatGapDuration(ttlSeconds)}</Text>
                    {status && <Text size="xs" c={color}>{status}</Text>}
                </Stack>
            }
            withArrow
            multiline
            w={230}
        >
            {label}
        </Tooltip>
    );
}

function sourceBadgeColor(source: string): string {
    switch (source) {
        case "heartbeat":
            return "orange";
        case "manual":
            return "grape";
        case "expiry":
            return "orange";
        case "limit_exceeded":
            return "red";
        default:
            return "gray";
    }
}

export interface AddressHistoryColumnDeps {
    formatDateTime: (value: string) => string;
    pickerValueFormat: string;

    presetStr: string | null;
    fromStr: string | null;
    toStr: string | null;
    ipLocal: string;
    ipDebounced: string;
    deviceIdStr: string | null;
    sourceStr: string | null;
    enabledStr: string | null;
    lockedDeviceId?: number;

    deviceOptions: { value: string; label: string }[];

    setParam: (key: string, value: string | null) => void;
    setIpLocal: (value: string) => void;
    setSearchParams: AddressHistoryFilters["setSearchParams"];
    onDeviceClick: (deviceId: number) => void;
}

export function getAddressHistoryColumns(deps: AddressHistoryColumnDeps): DataTableColumn<AddressHistoryEvent>[] {
    const sourceOptions = Object.entries(SOURCE_LABELS).map(([value, label]) => ({ value, label }));

    const allColumns: DataTableColumn<AddressHistoryEvent>[] = [
        {
            accessor: "timestamp",
            title: "Time",
            filter: () => (
                <Stack gap="xs" p="xs">
                    <DateTimePicker
                        label="From"
                        placeholder="24 hours ago"
                        value={deps.fromStr ?? null}
                        maxDate={deps.toStr ?? undefined}
                        getDayProps={(date) =>
                            deps.toStr?.startsWith(date) ? { style: refDateStyle } : {}
                        }
                        onChange={(val) => {
                            deps.setSearchParams((prev) => {
                                prev.delete("preset");
                                if (val) prev.set("from", dayjs(val).toISOString());
                                else prev.delete("from");
                                return prev;
                            });
                        }}
                        valueFormat={deps.pickerValueFormat}
                        highlightToday
                        timePickerProps={{
                            withDropdown: true,
                            popoverProps: { withinPortal: false },
                        }}
                        popoverProps={{ withinPortal: false }}
                        clearable
                        w={280}
                    />
                    <DateTimePicker
                        label="To"
                        placeholder="Now (live)"
                        value={deps.toStr ?? null}
                        minDate={deps.fromStr ?? undefined}
                        getDayProps={(date) =>
                            deps.fromStr?.startsWith(date) ? { style: refDateStyle } : {}
                        }
                        onChange={(val) => {
                            deps.setSearchParams((prev) => {
                                prev.delete("preset");
                                if (val) prev.set("to", dayjs(val).toISOString());
                                else prev.delete("to");
                                return prev;
                            });
                        }}
                        valueFormat={deps.pickerValueFormat}
                        highlightToday
                        timePickerProps={{
                            withDropdown: true,
                            popoverProps: { withinPortal: false },
                        }}
                        popoverProps={{ withinPortal: false }}
                        clearable
                        w={280}
                    />
                </Stack>
            ),
            filtering: !!(deps.fromStr || deps.toStr),
            render: (row) => (
                <Text size="sm" ff="monospace">
                    {deps.formatDateTime(row.timestamp)}
                </Text>
            ),
        },
        {
            accessor: "time_gap_seconds",
            title: "Δ prev",
            render: (row) => renderGapCell({ gapSeconds: row.time_gap_seconds, ttlSeconds: row.ttl_seconds }),
        },
        {
            accessor: "device_name",
            title: "Device",
            filter: ({ close }) => (
                <Select
                    placeholder="All devices"
                    data={deps.deviceOptions}
                    value={deps.deviceIdStr}
                    onChange={(val) => { deps.setParam("device_id", val); close(); }}
                    clearable
                    comboboxProps={{ withinPortal: false }}
                    m="xs"
                    w={200}
                />
            ),
            filtering: !!deps.deviceIdStr,
            render: (row) => (
                <Anchor size="sm" onClick={() => deps.onDeviceClick(row.device_id)}>
                    {row.device_name}
                </Anchor>
            ),
        },
        {
            accessor: "ip",
            title: "IP",
            filter: ({ close }) => (
                <TextInput
                    placeholder="Filter by IP"
                    leftSection={<IconSearch size={16} />}
                    value={deps.ipLocal}
                    onChange={(e) => deps.setIpLocal(e.currentTarget.value)}
                    onKeyDown={(e) => { if (e.key === "Enter") close(); }}
                    m="xs"
                    w={200}
                />
            ),
            filtering: !!deps.ipDebounced,
            render: (row) => (
                <Group gap={6} wrap="nowrap">
                    <Text size="sm" ff="monospace">
                        {row.ip}
                    </Text>
                    {row.ip_changed && (
                        <Tooltip label="IP changed from this device's previous event" withArrow>
                            <ThemeIcon size="xs" variant="transparent" color="indigo">
                                <IconArrowsRightLeft size={12} />
                            </ThemeIcon>
                        </Tooltip>
                    )}
                </Group>
            ),
        },
        {
            accessor: "is_enabled",
            title: "Status",
            filter: ({ close }) => (
                <SegmentedControl
                    data={[
                        { label: "All", value: "all" },
                        { label: "Enabled", value: "true" },
                        { label: "Disabled", value: "false" },
                    ]}
                    value={deps.enabledStr ?? "all"}
                    onChange={(val) => {
                        deps.setParam("is_enabled", val === "all" ? null : val);
                        close();
                    }}
                    m="xs"
                />
            ),
            filtering: !!deps.enabledStr,
            render: (row) => (
                <Badge
                    color={row.is_enabled ? "green" : "red"}
                    size="sm"
                    variant="light"
                >
                    {row.is_enabled ? "Enabled" : "Disabled"}
                </Badge>
            ),
        },
        {
            accessor: "source",
            title: "Source",
            filter: ({ close }) => (
                <Select
                    placeholder="All sources"
                    data={sourceOptions}
                    value={deps.sourceStr}
                    onChange={(val) => { deps.setParam("source", val); close(); }}
                    clearable
                    comboboxProps={{ withinPortal: false }}
                    m="xs"
                    w={200}
                />
            ),
            filtering: !!deps.sourceStr,
            render: (row) => (
                <Badge
                    size="sm"
                    color={sourceBadgeColor(row.source)}
                    variant="dot"
                >
                    {SOURCE_LABELS[row.source] ?? row.source}
                </Badge>
            ),
        },
    ];

    if (deps.lockedDeviceId != null) {
        return allColumns.filter((c) => c.accessor !== "device_name");
    }
    return allColumns;
}
