import {
    ActionIcon,
    Anchor,
    Badge,
    Group,
    SegmentedControl,
    Select,
    Stack,
    Text,
    TextInput,
} from "@mantine/core";
import { DateTimePicker } from "@mantine/dates";
import { IconChevronRight, IconHome, IconHexagon, IconSearch } from "@tabler/icons-react";
import type { DataTableColumn } from "mantine-datatable";
import type { AccessLogRow } from "@/lib/api";
import type { AccessLogFilters } from "../hooks/useAccessLogFilters";
import { DENY_REASON_LABELS } from "../constants";
import { countryFlagEmoji } from "@/lib/countryFlag";
import dayjs from "dayjs";

/** Style for marking the "other" endpoint date on each picker's calendar. */
const refDateStyle = {
    outline: "1.5px dashed var(--mantine-color-dimmed)",
    outlineOffset: -1.5,
    borderRadius: "var(--mantine-radius-sm)",
} as const;

export interface AccessLogColumnDeps {
    formatDateTime: (value: string) => string;
    pickerValueFormat: string;

    // Filter values
    presetStr: string | null;
    fromStr: string | null;
    toStr: string | null;
    ipLocal: string;
    ipDebounced: string;
    deviceIdStr: string | null;
    networkPolicyIdStr: string | null;
    outcomeStr: string | null;
    denyReason: string | null;
    countryCodeLocal: string;
    countryCodeDebounced: string;

    // Options
    deviceOptions: { value: string; label: string }[];
    denyReasonOptions: { value: string; label: string }[];
    networkPolicyOptions: { value: string; label: string }[];

    // Setters
    setParam: (key: string, value: string | null) => void;
    setNetworkPolicyId: (val: string | null) => void;
    setIpLocal: (value: string) => void;
    setCountryCodeLocal: (value: string) => void;
    setSearchParams: AccessLogFilters["setSearchParams"];

    // Actions
    onRowClick: (row: AccessLogRow) => void;
    onDeviceClick: (deviceId: number) => void;
    onNetworkPolicyClick: (id: number) => void;
}

export function getAccessLogColumns(deps: AccessLogColumnDeps): DataTableColumn<AccessLogRow>[] {
    return [
        {
            accessor: "created_at",
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
                    {deps.formatDateTime(row.created_at)}
                </Text>
            ),
        },
        {
            accessor: "client_ip",
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
                    {row.client_ip}
                </Text>
            ),
        },
        {
            accessor: "country_code",
            title: "Country",
            filter: ({ close }) => (
                <TextInput
                    placeholder="e.g. DE"
                    leftSection={<IconSearch size={16} />}
                    value={deps.countryCodeLocal}
                    onChange={(e) => deps.setCountryCodeLocal(e.currentTarget.value.toUpperCase())}
                    onKeyDown={(e) => { if (e.key === "Enter") close(); }}
                    m="xs"
                    w={160}
                />
            ),
            filtering: !!deps.countryCodeDebounced,
            render: (row) =>
                row.country_code ? (
                    <Text size="sm">
                        {countryFlagEmoji(row.country_code)} {row.country_code}
                    </Text>
                ) : (
                    <IconHome size={14} color="var(--mantine-color-dimmed)" />
                ),
        },
        {
            accessor: "target_host",
            title: "Host",
            render: (row) => (
                <Text size="sm">{row.target_host ?? "—"}</Text>
            ),
        },
        {
            accessor: "device_name",
            title: "Authorized by",
            filter: ({ close }) => (
                <Stack gap="xs" p="xs">
                    <Text size="xs" c="dimmed" fw={500}>By device</Text>
                    <Select
                        placeholder="All devices"
                        data={deps.deviceOptions}
                        value={deps.deviceIdStr}
                        onChange={(val) => {
                            deps.setParam("network_policy_id", null);
                            deps.setParam("device_id", val);
                            close();
                        }}
                        clearable
                        comboboxProps={{ withinPortal: false }}
                        w={220}
                    />
                    <Text size="xs" c="dimmed" fw={500}>By network policy</Text>
                    <Select
                        placeholder="All policies"
                        data={deps.networkPolicyOptions}
                        value={deps.networkPolicyIdStr}
                        onChange={(val) => { deps.setNetworkPolicyId(val); close(); }}
                        clearable
                        comboboxProps={{ withinPortal: false }}
                        w={220}
                    />
                </Stack>
            ),
            filtering: !!(deps.deviceIdStr || deps.networkPolicyIdStr),
            render: (row) => {
                if (row.network_policy_id != null) {
                    return (
                        <Anchor
                            size="sm"
                            c="teal.5"
                            onClick={(e) => { e.stopPropagation(); deps.onNetworkPolicyClick(row.network_policy_id!); }}
                        >
                            <Group gap={4} wrap="nowrap">
                                <IconHexagon size={14} />
                                <Text size="sm" inherit>{row.network_policy_name ?? "Unknown policy"}</Text>
                            </Group>
                        </Anchor>
                    );
                }
                if (row.device_name && row.device_id != null) {
                    return (
                        <Anchor
                            size="sm"
                            onClick={(e) => { e.stopPropagation(); deps.onDeviceClick(row.device_id!); }}
                        >
                            {row.device_name}
                        </Anchor>
                    );
                }
                return <Text size="sm" c="dimmed">—</Text>;
            },
        },
        {
            accessor: "outcome",
            title: "Outcome",
            filter: ({ close }) => (
                <SegmentedControl
                    data={[
                        { label: "All", value: "all" },
                        { label: "Allow", value: "allow" },
                        { label: "Deny", value: "deny" },
                    ]}
                    value={deps.outcomeStr ?? "all"}
                    onChange={(val) => {
                        if (val === "all") {
                            deps.setSearchParams((prev) => {
                                prev.delete("outcome");
                                return prev;
                            });
                        } else if (val === "allow") {
                            deps.setSearchParams((prev) => {
                                prev.set("outcome", val);
                                prev.delete("deny_reason");
                                return prev;
                            });
                        } else {
                            deps.setParam("outcome", val);
                        }
                        close();
                    }}
                    m="xs"
                />
            ),
            filtering: !!deps.outcomeStr,
            render: (row) => (
                <Badge color={row.outcome ? "green" : "red"} size="sm">
                    {row.outcome ? "Allow" : "Deny"}
                </Badge>
            ),
        },
        {
            accessor: "deny_reason",
            title: "Reason",
            filter: ({ close }) => (
                <Select
                    placeholder="Any reason"
                    data={deps.denyReasonOptions}
                    value={deps.denyReason}
                    onChange={(val) => { deps.setParam("deny_reason", val); close(); }}
                    clearable
                    comboboxProps={{ withinPortal: false }}
                    m="xs"
                    w={200}
                />
            ),
            filtering: !!deps.denyReason,
            render: (row) =>
                row.deny_reason ? (
                    <Text size="sm">
                        {DENY_REASON_LABELS[row.deny_reason] ?? row.deny_reason}
                    </Text>
                ) : (
                    <Text size="sm" c="dimmed">
                        —
                    </Text>
                ),
        },
        {
            accessor: "duration_us",
            title: "Duration",
            render: (row) => (
                <Text size="sm" ff="monospace">
                    {row.duration_us != null ? `${(row.duration_us / 1000).toFixed(2)} ms` : "—"}
                </Text>
            ),
        },
        {
            accessor: "actions",
            title: "",
            width: 48,
            render: (row) => (
                <ActionIcon
                    variant="subtle"
                    color="gray"
                    size="md"
                    onClick={() => deps.onRowClick(row)}
                    aria-label="View details"
                >
                    <IconChevronRight size={18} />
                </ActionIcon>
            ),
        },
    ];
}
