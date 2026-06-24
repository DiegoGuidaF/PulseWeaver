import {
    ActionIcon,
    Anchor,
    Badge,
    Group,
    SegmentedControl,
    Stack,
    Text,
    Tooltip,
    VisuallyHidden,
} from "@mantine/core";
import { DateTimePicker } from "@mantine/dates";
import { IconChevronRight, IconHome, IconHexagon } from "@tabler/icons-react";
import type { DataTableColumn } from "mantine-datatable";
import { FilterableCell } from "./FilterableCell";
import type { AccessLogContributor, AccessLogRow } from "@/lib/api";
import type { AccessLogFilters } from "../hooks/useAccessLogFilters";
import {
    type ColumnFilterState,
    type FilterColumnKey,
    FILTER_COLUMNS,
    HTTP_METHODS,
    isFilterActive,
} from "../filterConfig";
import { ColumnFilter, FilterApplyButton } from "./ColumnFilter";
import { POLICY_DENY_REASON_LABELS } from "@/lib/policyDenyReasons";
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

    // Time window
    fromStr: string | null;
    toStr: string | null;

    // Outcome
    outcomeStr: string | null;
    setOutcome: (value: string | null) => void;

    // Generic column filters
    getColumnFilter: (key: FilterColumnKey) => ColumnFilterState;
    setColumnFilter: (key: FilterColumnKey, state: ColumnFilterState | null) => void;
    setSearchParams: AccessLogFilters["setSearchParams"];

    // Options
    deviceOptions: { value: string; label: string }[];
    denyReasonOptions: { value: string; label: string }[];
    networkPolicyOptions: { value: string; label: string }[];
    userOptions: { value: string; label: string }[];

    // Actions
    onRowClick: (row: AccessLogRow) => void;
    onDeviceClick: (deviceId: number, ownerUserId: number | undefined) => void;
    onUserClick: (userId: number) => void;
    onNetworkPolicyClick: (id: number) => void;
}

/** Distinct contributing users (by user_id), in first-seen order. */
function distinctUsers(contributors: AccessLogContributor[]): AccessLogContributor[] {
    const seen = new Set<number>();
    const out: AccessLogContributor[] = [];
    for (const c of contributors) {
        if (c.user_id == null || seen.has(c.user_id)) continue;
        seen.add(c.user_id);
        out.push(c);
    }
    return out;
}

/** Distinct contributing devices (by device_id), in first-seen order. */
function distinctDevices(contributors: AccessLogContributor[]): AccessLogContributor[] {
    const seen = new Set<number>();
    const out: AccessLogContributor[] = [];
    for (const c of contributors) {
        if (c.device_id == null || seen.has(c.device_id)) continue;
        seen.add(c.device_id);
        out.push(c);
    }
    return out;
}

function overflowBadge(count: number, contributorCount: number) {
    if (count <= 1) return null;
    return (
        <Tooltip label={`Client IP resolved to ${contributorCount} contributors`}>
            <Badge size="xs" variant="light" color="gray" style={{ flexShrink: 0 }}>
                +{count - 1}
            </Badge>
        </Tooltip>
    );
}

export function getAccessLogColumns(deps: AccessLogColumnDeps): DataTableColumn<AccessLogRow>[] {
    const columnFilterSlot =
        (key: FilterColumnKey, options?: { value: string; label: string }[]) =>
        ({ close }: { close: () => void }) => (
            <Stack gap="xs" p="xs">
                <ColumnFilter
                    config={FILTER_COLUMNS[key]}
                    state={deps.getColumnFilter(key)}
                    options={options}
                    onCommit={(next) => deps.setColumnFilter(key, next)}
                />
                <FilterApplyButton onApply={close} />
            </Stack>
        );

    return [
        {
            accessor: "created_at",
            title: "Time",
            sortable: true,
            filter: () => (
                <Stack gap="xs" p="xs">
                    <DateTimePicker
                        data-autofocus
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
            sortable: true,
            filter: columnFilterSlot("client_ip"),
            filtering: isFilterActive(deps.getColumnFilter("client_ip")),
            render: (row) => (
                <FilterableCell
                    filterLabel="Filter by this IP"
                    onFilter={() =>
                        deps.setColumnFilter("client_ip", { op: "in", values: [row.client_ip] })
                    }
                >
                    <Text size="sm" ff="monospace" truncate>
                        {row.client_ip}
                    </Text>
                </FilterableCell>
            ),
        },
        {
            accessor: "country_code",
            title: "Country",
            filter: columnFilterSlot("country_code"),
            filtering: isFilterActive(deps.getColumnFilter("country_code")),
            render: (row) =>
                row.country_code ? (
                    <FilterableCell
                        filterLabel="Filter by this country"
                        onFilter={() =>
                            deps.setColumnFilter("country_code", { op: "in", values: [row.country_code!] })
                        }
                    >
                        <Text size="sm">
                            {countryFlagEmoji(row.country_code)} {row.country_code}
                        </Text>
                    </FilterableCell>
                ) : (
                    <IconHome size={14} color="var(--mantine-color-dimmed)" />
                ),
        },
        {
            accessor: "target_host",
            title: "Host",
            sortable: true,
            width: 180,
            ellipsis: true,
            filter: columnFilterSlot("target_host"),
            filtering: isFilterActive(deps.getColumnFilter("target_host")),
            render: (row) =>
                row.target_host ? (
                    <FilterableCell
                        filterLabel="Filter by this host"
                        onFilter={() =>
                            deps.setColumnFilter("target_host", { op: "in", values: [row.target_host!] })
                        }
                    >
                        <Text size="sm" truncate title={row.target_host}>
                            {row.target_host}
                        </Text>
                    </FilterableCell>
                ) : (
                    <Text size="sm" c="dimmed">—</Text>
                ),
        },
        {
            accessor: "target_uri",
            title: "URI",
            width: 280,
            ellipsis: true,
            filter: columnFilterSlot("target_uri"),
            filtering: isFilterActive(deps.getColumnFilter("target_uri")),
            render: (row) => (
                <Text size="sm" ff="monospace" truncate title={row.target_uri ?? undefined}>
                    {row.target_uri ?? "—"}
                </Text>
            ),
        },
        {
            accessor: "http_method",
            title: "Method",
            sortable: true,
            textAlign: "center",
            filter: columnFilterSlot(
                "http_method",
                HTTP_METHODS.map((m) => ({ value: m, label: m })),
            ),
            filtering: isFilterActive(deps.getColumnFilter("http_method")),
            render: (row) => <Text size="sm" ff="monospace">{row.http_method ?? "—"}</Text>,
        },
        {
            accessor: "user_id",
            title: "User",
            filter: columnFilterSlot("user_id", deps.userOptions),
            filtering: isFilterActive(deps.getColumnFilter("user_id")),
            render: (row) => {
                const users = distinctUsers(row.contributors);
                if (users.length === 0) return <Text size="sm" c="dimmed">—</Text>;
                const first = users[0];
                return (
                    <FilterableCell
                        filterLabel="Filter by this user"
                        onFilter={
                            first.user_id != null
                                ? () =>
                                      deps.setColumnFilter("user_id", {
                                          op: "in",
                                          values: [String(first.user_id)],
                                      })
                                : undefined
                        }
                    >
                        <Group gap={6} wrap="nowrap" style={{ minWidth: 0 }}>
                            <Anchor
                                size="sm"
                                truncate
                                onClick={(e) => {
                                    e.stopPropagation();
                                    if (first.user_id != null) deps.onUserClick(first.user_id);
                                }}
                            >
                                {first.user_name ?? `User #${first.user_id}`}
                            </Anchor>
                            {overflowBadge(users.length, row.contributor_count)}
                        </Group>
                    </FilterableCell>
                );
            },
        },
        {
            accessor: "authorized_by",
            title: "Authorized by",
            filter: ({ close }) => (
                <Stack gap="sm" p="xs">
                    <Stack gap={4}>
                        <Text size="xs" c="dimmed" fw={500}>By device</Text>
                        <ColumnFilter
                            config={FILTER_COLUMNS.device_id}
                            state={deps.getColumnFilter("device_id")}
                            options={deps.deviceOptions}
                            onCommit={(next) => deps.setColumnFilter("device_id", next)}
                            autoFocus={false}
                        />
                    </Stack>
                    <Stack gap={4}>
                        <Text size="xs" c="dimmed" fw={500}>By network policy</Text>
                        <ColumnFilter
                            config={FILTER_COLUMNS.network_policy_id}
                            state={deps.getColumnFilter("network_policy_id")}
                            options={deps.networkPolicyOptions}
                            onCommit={(next) => deps.setColumnFilter("network_policy_id", next)}
                            autoFocus={false}
                        />
                    </Stack>
                    <FilterApplyButton onApply={close} />
                </Stack>
            ),
            filtering:
                isFilterActive(deps.getColumnFilter("device_id")) ||
                isFilterActive(deps.getColumnFilter("network_policy_id")),
            render: (row) => {
                if (row.network_policy_id != null) {
                    return (
                        <FilterableCell
                            filterLabel="Filter by this network policy"
                            onFilter={() =>
                                deps.setColumnFilter("network_policy_id", {
                                    op: "in",
                                    values: [String(row.network_policy_id)],
                                })
                            }
                        >
                            <Anchor
                                size="sm"
                                c="teal.5"
                                onClick={(e) => {
                                    e.stopPropagation();
                                    deps.onNetworkPolicyClick(row.network_policy_id!);
                                }}
                            >
                                <Group gap={4} wrap="nowrap" style={{ minWidth: 0 }}>
                                    <IconHexagon size={14} style={{ flexShrink: 0 }} />
                                    <Text size="sm" inherit truncate>{row.network_policy_name ?? "Unknown policy"}</Text>
                                </Group>
                            </Anchor>
                        </FilterableCell>
                    );
                }
                const devices = distinctDevices(row.contributors);
                if (devices.length === 0) return <Text size="sm" c="dimmed">—</Text>;
                const first = devices[0];
                return (
                    <FilterableCell
                        filterLabel="Filter by this device"
                        onFilter={
                            first.device_id != null
                                ? () =>
                                      deps.setColumnFilter("device_id", {
                                          op: "in",
                                          values: [String(first.device_id)],
                                      })
                                : undefined
                        }
                    >
                        <Group gap={6} wrap="nowrap" style={{ minWidth: 0 }}>
                            {first.user_id != null ? (
                                <Anchor
                                    size="sm"
                                    truncate
                                    onClick={(e) => {
                                        e.stopPropagation();
                                        deps.onDeviceClick(first.device_id!, first.user_id);
                                    }}
                                >
                                    {first.device_name ?? `Device #${first.device_id}`}
                                </Anchor>
                            ) : (
                                <Text size="sm" truncate>{first.device_name ?? `Device #${first.device_id}`}</Text>
                            )}
                            {overflowBadge(devices.length, row.contributor_count)}
                        </Group>
                    </FilterableCell>
                );
            },
        },
        {
            accessor: "outcome",
            title: "Decision",
            sortable: true,
            filter: ({ close }) => (
                <Stack gap="sm" p="xs">
                    <Stack gap={4}>
                        <Text size="xs" c="dimmed" fw={500}>Outcome</Text>
                        <SegmentedControl
                            fullWidth
                            data={[
                                { label: "All", value: "all" },
                                { label: "Allow", value: "allow" },
                                { label: "Deny", value: "deny" },
                            ]}
                            value={deps.outcomeStr ?? "all"}
                            onChange={(val) => deps.setOutcome(val === "all" ? null : val)}
                        />
                    </Stack>
                    <Stack gap={4}>
                        <Text size="xs" c="dimmed" fw={500}>Deny reason</Text>
                        <ColumnFilter
                            config={FILTER_COLUMNS.deny_reason}
                            state={deps.getColumnFilter("deny_reason")}
                            options={deps.denyReasonOptions}
                            onCommit={(next) => deps.setColumnFilter("deny_reason", next)}
                            autoFocus={false}
                        />
                    </Stack>
                    <FilterApplyButton onApply={close} />
                </Stack>
            ),
            filtering: !!deps.outcomeStr || isFilterActive(deps.getColumnFilter("deny_reason")),
            render: (row) => (
                <FilterableCell
                    filterLabel={`Filter by ${row.outcome ? "allowed" : "denied"} requests`}
                    onFilter={() => deps.setOutcome(row.outcome ? "allow" : "deny")}
                >
                    <Group gap="xs" wrap="nowrap" style={{ minWidth: 0 }}>
                        <Badge color={row.outcome ? "green" : "red"} size="sm" style={{ flexShrink: 0 }}>
                            {row.outcome ? "Allow" : "Deny"}
                        </Badge>
                        {!row.outcome && row.deny_reason && (
                            <Text size="sm" c="dimmed" truncate title={POLICY_DENY_REASON_LABELS[row.deny_reason]}>
                                {POLICY_DENY_REASON_LABELS[row.deny_reason]}
                            </Text>
                        )}
                    </Group>
                </FilterableCell>
            ),
        },
        {
            accessor: "duration_us",
            title: "Duration",
            sortable: true,
            textAlign: "right",
            render: (row) => (
                <Text size="sm" ff="monospace">
                    {row.duration_us != null ? `${(row.duration_us / 1000).toFixed(2)} ms` : "—"}
                </Text>
            ),
        },
        {
            accessor: "actions",
            title: <VisuallyHidden>Actions</VisuallyHidden>,
            width: 48,
            render: (row) => (
                <Tooltip label="View details" position="left" withArrow>
                    <ActionIcon
                        variant="subtle"
                        size="md"
                        onClick={() => deps.onRowClick(row)}
                        aria-label="View details"
                    >
                        <IconChevronRight size={18} />
                    </ActionIcon>
                </Tooltip>
            ),
        },
    ];
}
