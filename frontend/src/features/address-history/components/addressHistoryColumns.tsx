import {
    Badge,
    SegmentedControl,
    Select,
    Stack,
    Text,
    TextInput,
} from "@mantine/core";
import { DateTimePicker } from "@mantine/dates";
import { IconSearch } from "@tabler/icons-react";
import type { DataTableColumn } from "mantine-datatable";
import type { AddressHistoryEvent } from "@/lib/api";
import type { AddressHistoryFilters } from "../hooks/useAddressHistoryFilters";
import { SOURCE_LABELS } from "../constants";
import dayjs from "dayjs";

const refDateStyle = {
    outline: "1.5px dashed var(--mantine-color-dimmed)",
    outlineOffset: -1.5,
    borderRadius: "var(--mantine-radius-sm)",
} as const;

function sourceBadgeColor(source: string): string {
    switch (source) {
        case "heartbeat":
            return "blue";
        case "manual":
            return "grape";
        case "expiry":
            return "orange";
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

    deviceOptions: { value: string; label: string }[];

    setParam: (key: string, value: string | null) => void;
    setIpLocal: (value: string) => void;
    setSearchParams: AddressHistoryFilters["setSearchParams"];
}

export function getAddressHistoryColumns(deps: AddressHistoryColumnDeps): DataTableColumn<AddressHistoryEvent>[] {
    const sourceOptions = Object.entries(SOURCE_LABELS).map(([value, label]) => ({ value, label }));

    return [
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
                <Text size="sm">{row.device_name}</Text>
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
                <Text size="sm" ff="monospace">
                    {row.ip}
                </Text>
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
}
