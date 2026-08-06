import { Anchor, Badge, Group, Stack, Text, ThemeIcon, Tooltip } from "@mantine/core";
import { DateTimePicker } from "@mantine/dates";
import { IconArrowsRightLeft } from "@tabler/icons-react";
import type { DataTableColumn } from "mantine-datatable";
import { FilterableCell } from "@/components/FilterableCell";
import { ColumnFilter, FilterApplyButton } from "@/components/ColumnFilter";
import { GeoCell } from "@/components/GeoCell";
import { AddressEventSource, AddressHistoryFilterOperator, type AddressEventKind, type AddressHistoryEvent, type TtlRisk } from "@/lib/api";
import type { AddressHistoryFilters } from "../hooks/useAddressHistoryFilters";
import {
    type ColumnFilterState,
    type FilterColumnKey,
    FILTER_COLUMNS,
    isFilterActive,
} from "../filterConfig";
import {
    EVENT_KIND_COLORS,
    EVENT_KIND_LABELS,
    EVENT_KIND_OPTIONS,
    SOURCE_LABELS,
    SOURCE_OPTIONS,
    TTL_RISK_BADGE,
    TTL_RISK_LABELS,
    TTL_RISK_OPTIONS,
    formatGapDuration,
} from "../constants";
import dayjs from "dayjs";

const refDateStyle = {
    outline: "1.5px dashed var(--mantine-color-dimmed)",
    outlineOffset: -1.5,
    borderRadius: "var(--mantine-radius-sm)",
} as const;

function renderGapCell(row: AddressHistoryEvent) {
    if (row.renewal_gap_seconds == null) {
        return <Text size="sm" ff="monospace" c="dimmed">—</Text>;
    }

    const label = (
        <Text size="sm" ff="monospace">
            {formatGapDuration(row.renewal_gap_seconds)}
        </Text>
    );

    if (row.ttl_seconds == null) return label;

    return (
        <Tooltip label={`Device TTL: ${formatGapDuration(row.ttl_seconds)}`} withArrow>
            {label}
        </Tooltip>
    );
}

function sourceBadgeColor(source: AddressEventSource): string {
    switch (source) {
        case AddressEventSource.HEARTBEAT:
            return "orange";
        case AddressEventSource.MANUAL:
            return "grape";
        case AddressEventSource.EXPIRY:
            return "orange";
        case AddressEventSource.LIMIT_EXCEEDED:
            return "red";
        default: {
            const _exhaustive: never = source;
            return _exhaustive;
        }
    }
}

function renderEventKindBadge(kind: AddressEventKind) {
    return (
        <Badge size="sm" color={EVENT_KIND_COLORS[kind]} variant="dot">
            {EVENT_KIND_LABELS[kind]}
        </Badge>
    );
}

function renderTtlRiskBadge(risk: TtlRisk) {
    const { color, variant } = TTL_RISK_BADGE[risk];
    return (
        <Badge size="sm" color={color} variant={variant}>
            {TTL_RISK_LABELS[risk]}
        </Badge>
    );
}

/** Column(s) to hide when a given filter is locked — locking implies every row already shares that value, so displaying it is redundant. */
const HIDDEN_COLUMNS_WHEN_LOCKED: Partial<Record<FilterColumnKey, string[]>> = {
    device_id: ["device_name"],
};

export interface AddressHistoryColumnDeps {
    formatDateTime: (value: string) => string;
    pickerValueFormat: string;

    fromStr: string | null;
    toStr: string | null;
    setSearchParams: AddressHistoryFilters["setSearchParams"];

    getColumnFilter: (key: FilterColumnKey) => ColumnFilterState;
    setColumnFilter: (key: FilterColumnKey, state: ColumnFilterState | null) => void;
    lockedFilterKey?: FilterColumnKey;

    deviceOptions: { value: string; label: string }[];
    userOptions: { value: string; label: string }[];

    onDeviceClick: (deviceId: number) => void;
}

export function getAddressHistoryColumns(deps: AddressHistoryColumnDeps): DataTableColumn<AddressHistoryEvent>[] {
    const columnFilterSlot =
        (key: FilterColumnKey, options?: { value: string; label: string }[], placeholder?: string) =>
        ({ close }: { close: () => void }) => (
            <Stack gap="xs" p="xs">
                <ColumnFilter
                    config={FILTER_COLUMNS[key]}
                    state={deps.getColumnFilter(key)}
                    options={options}
                    placeholder={placeholder}
                    onCommit={(next) => deps.setColumnFilter(key, next)}
                />
                <FilterApplyButton onApply={close} />
            </Stack>
        );

    const allColumns: DataTableColumn<AddressHistoryEvent>[] = [
        {
            accessor: "timestamp",
            title: "Time",
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
                    {deps.formatDateTime(row.timestamp)}
                </Text>
            ),
        },
        {
            accessor: "renewal_gap_seconds",
            title: "Renewal gap",
            textAlign: "right",
            render: renderGapCell,
        },
        {
            accessor: "device_name",
            title: "Device",
            filter: ({ close }) => (
                <Stack gap="sm" p="xs">
                    {deps.lockedFilterKey !== "device_id" && (
                        <Stack gap={4}>
                            <Text size="xs" c="dimmed" fw={500}>By device</Text>
                            <ColumnFilter
                                config={FILTER_COLUMNS.device_id}
                                state={deps.getColumnFilter("device_id")}
                                options={deps.deviceOptions}
                                onCommit={(next) => deps.setColumnFilter("device_id", next)}
                            />
                        </Stack>
                    )}
                    {deps.lockedFilterKey !== "user_id" && (
                        <Stack gap={4}>
                            <Text size="xs" c="dimmed" fw={500}>By owning user</Text>
                            <ColumnFilter
                                config={FILTER_COLUMNS.user_id}
                                state={deps.getColumnFilter("user_id")}
                                options={deps.userOptions}
                                onCommit={(next) => deps.setColumnFilter("user_id", next)}
                                autoFocus={false}
                            />
                        </Stack>
                    )}
                    <FilterApplyButton onApply={close} />
                </Stack>
            ),
            filtering:
                isFilterActive(deps.getColumnFilter("device_id")) ||
                isFilterActive(deps.getColumnFilter("user_id")),
            render: (row) => (
                <FilterableCell
                    filterLabel="Filter by this device"
                    onFilter={
                        deps.lockedFilterKey === "device_id"
                            ? undefined
                            : () =>
                                  deps.setColumnFilter("device_id", {
                                      op: AddressHistoryFilterOperator.IN,
                                      values: [String(row.device_id)],
                                  })
                    }
                >
                    <Anchor size="sm" onClick={() => deps.onDeviceClick(row.device_id)}>
                        {row.device_name}
                    </Anchor>
                </FilterableCell>
            ),
        },
        {
            accessor: "ip",
            title: "IP",
            filter: columnFilterSlot("ip", undefined, "Filter by IP"),
            filtering: isFilterActive(deps.getColumnFilter("ip")),
            render: (row) => (
                <FilterableCell
                    filterLabel="Filter by this IP"
                    onFilter={() =>
                        deps.setColumnFilter("ip", { op: AddressHistoryFilterOperator.IN, values: [row.ip] })
                    }
                >
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
                </FilterableCell>
            ),
        },
        {
            accessor: "geo",
            title: "Location",
            render: (row) => <GeoCell geo={row.geo} size="sm" />,
        },
        {
            accessor: "is_enabled",
            title: "Status",
            textAlign: "center",
            render: (row) => (
                <Badge color={row.is_enabled ? "green" : "red"} size="sm" variant="light">
                    {row.is_enabled ? "Enabled" : "Disabled"}
                </Badge>
            ),
        },
        {
            accessor: "source",
            title: "Source",
            textAlign: "center",
            filter: columnFilterSlot("source", SOURCE_OPTIONS),
            filtering: isFilterActive(deps.getColumnFilter("source")),
            render: (row) => (
                <FilterableCell
                    filterLabel="Filter by this source"
                    onFilter={() =>
                        deps.setColumnFilter("source", { op: AddressHistoryFilterOperator.IN, values: [row.source] })
                    }
                >
                    <Badge size="sm" color={sourceBadgeColor(row.source)} variant="dot">
                        {SOURCE_LABELS[row.source] ?? row.source}
                    </Badge>
                </FilterableCell>
            ),
        },
        {
            accessor: "event_kind",
            title: "Event",
            textAlign: "center",
            filter: columnFilterSlot("event_kind", EVENT_KIND_OPTIONS),
            filtering: isFilterActive(deps.getColumnFilter("event_kind")),
            render: (row) => (
                <FilterableCell
                    filterLabel="Filter by this event kind"
                    onFilter={() =>
                        deps.setColumnFilter("event_kind", {
                            op: AddressHistoryFilterOperator.IN,
                            values: [row.event_kind],
                        })
                    }
                >
                    {renderEventKindBadge(row.event_kind)}
                </FilterableCell>
            ),
        },
        {
            accessor: "ttl_risk",
            title: "TTL risk",
            textAlign: "center",
            filter: columnFilterSlot("ttl_risk", TTL_RISK_OPTIONS),
            filtering: isFilterActive(deps.getColumnFilter("ttl_risk")),
            render: (row) => (
                <FilterableCell
                    filterLabel="Filter by this TTL risk"
                    onFilter={() =>
                        deps.setColumnFilter("ttl_risk", {
                            op: AddressHistoryFilterOperator.IN,
                            values: [row.ttl_risk],
                        })
                    }
                >
                    {renderTtlRiskBadge(row.ttl_risk)}
                </FilterableCell>
            ),
        },
    ];

    const hidden = deps.lockedFilterKey ? (HIDDEN_COLUMNS_WHEN_LOCKED[deps.lockedFilterKey] ?? []) : [];
    return allColumns.filter((c) => !hidden.includes(String(c.accessor)));
}
