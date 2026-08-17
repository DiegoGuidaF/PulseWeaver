import { Anchor, Badge, Stack, Text, Tooltip } from "@mantine/core";
import { DateTimePicker } from "@mantine/dates";
import type { DataTableColumn } from "mantine-datatable";
import { FilterableCell } from "@/components/FilterableCell";
import { ColumnFilter, FilterApplyButton } from "@/components/ColumnFilter";
import { GeoCell } from "@/components/GeoCell";
import { AddressEventSource, AddressHistoryFilterOperator, TtlRisk, type AddressEventTrigger, type AddressHistoryEvent } from "@/lib/api";
import type { AddressHistoryFilters } from "../hooks/useAddressHistoryFilters";
import {
    type ColumnFilterState,
    type FilterColumnKey,
    FILTER_COLUMNS,
    isFilterActive,
} from "../filterConfig";
import {
    EVENT_KIND_BADGE_COLORS,
    EVENT_KIND_IMPLIES_ENABLED,
    EVENT_KIND_LABELS,
    EVENT_KIND_OPTIONS,
    TTL_HEADROOM_COLUMN_LABEL,
    SOURCE_LABELS,
    SOURCE_OPTIONS,
    TRIGGER_BADGE,
    TRIGGER_LABELS,
    TRIGGER_OPTIONS,
    TTL_RISK_BADGE,
    TTL_RISK_LABELS,
    TTL_RISK_OPTIONS,
    TTL_RISK_UNKNOWN_HINT,
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

/**
 * A column's usual value, stated without claiming any of the reader's attention.
 * Colour on this screen marks the exception; every column's modal value — which
 * is 86–93% of its rows — reads as quiet text so the exceptions can be seen.
 */
function quietText(label: string) {
    return (
        <Text span size="sm" c="dimmed">
            {label}
        </Text>
    );
}

function renderSourceCell(source: AddressEventSource) {
    const label = SOURCE_LABELS[source];
    // Which subsystem wrote an event only distinguishes anything on the rows the
    // Event badge has already stopped you at — measured, a disable is never a
    // heartbeat and a refresh is always one. So this is read, never scanned.
    return source === AddressEventSource.HEARTBEAT ? (
        quietText(label)
    ) : (
        <Text span size="sm">
            {label}
        </Text>
    );
}

function renderTriggerCell(trigger: AddressEventTrigger) {
    const badge = TRIGGER_BADGE[trigger];
    if (!badge) return quietText(TRIGGER_LABELS[trigger]);

    return (
        <Badge size="sm" color={badge.color} variant={badge.variant}>
            {TRIGGER_LABELS[trigger]}
        </Badge>
    );
}

function renderEventCell(row: AddressHistoryEvent) {
    const color = EVENT_KIND_BADGE_COLORS[row.event_kind];
    const label = EVENT_KIND_LABELS[row.event_kind];
    const kind = color ? (
        <Badge size="sm" color={color} variant="light">
            {label}
        </Badge>
    ) : (
        quietText(label)
    );

    // The kind implies the resulting address state on all but ~0.03% of events,
    // so stating it every time is a whole column of restatement. State it on the
    // rows where the implication breaks, which are the ones a reader would
    // otherwise misread — a `Created` row whose earlier history the retention
    // job has already pruned can be a disabled address.
    if (row.is_enabled === EVENT_KIND_IMPLIES_ENABLED[row.event_kind]) return kind;

    return (
        <>
            {kind}{" "}
            <Text span size="xs" c="dimmed">
                · now {row.is_enabled ? "enabled" : "disabled"}
            </Text>
        </>
    );
}

function renderTtlRiskCell(risk: TtlRisk) {
    // `unknown` is the absence of a health reading, not a fifth severity — a
    // grey "Not applicable" badge reads as a state the admin might need to act
    // on, so it degrades to the same dimmed dash the renewal gap uses.
    if (risk === TtlRisk.UNKNOWN) {
        return (
            <Tooltip label={TTL_RISK_UNKNOWN_HINT} withArrow>
                <Text span size="sm" ff="monospace" c="dimmed">
                    —
                </Text>
            </Tooltip>
        );
    }

    const badge = TTL_RISK_BADGE[risk];
    if (!badge) return quietText(TTL_RISK_LABELS[risk]);

    return (
        <Badge size="sm" color={badge.color} variant={badge.variant} c={badge.textColor}>
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
                    <Text size="sm" ff="monospace">
                        {row.ip}
                    </Text>
                </FilterableCell>
            ),
        },
        {
            accessor: "geo",
            title: "Location",
            render: (row) => <GeoCell geo={row.geo} size="sm" />,
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
                    {renderSourceCell(row.source)}
                </FilterableCell>
            ),
        },
        {
            accessor: "trigger_type",
            title: "Trigger",
            textAlign: "center",
            filter: columnFilterSlot("trigger_type", TRIGGER_OPTIONS),
            filtering: isFilterActive(deps.getColumnFilter("trigger_type")),
            render: (row) => (
                <FilterableCell
                    filterLabel="Filter by this trigger"
                    onFilter={() =>
                        deps.setColumnFilter("trigger_type", {
                            op: AddressHistoryFilterOperator.IN,
                            values: [row.trigger_type],
                        })
                    }
                >
                    {renderTriggerCell(row.trigger_type)}
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
                    // "Refresh" is 86% of this column and narrower than the
                    // badge the three transitions carry, so left to shrink the
                    // column sizes to the text and clips "Disabled" to "Dis…".
                    noShrink
                    onFilter={() =>
                        deps.setColumnFilter("event_kind", {
                            op: AddressHistoryFilterOperator.IN,
                            values: [row.event_kind],
                        })
                    }
                >
                    {renderEventCell(row)}
                </FilterableCell>
            ),
        },
        {
            accessor: "renewal_gap_seconds",
            title: "Renewal gap",
            textAlign: "right",
            render: renderGapCell,
        },
        {
            accessor: "ttl_risk",
            title: TTL_HEADROOM_COLUMN_LABEL,
            textAlign: "center",
            filter: columnFilterSlot("ttl_risk", TTL_RISK_OPTIONS),
            filtering: isFilterActive(deps.getColumnFilter("ttl_risk")),
            render: (row) => (
                <FilterableCell
                    filterLabel="Filter by this TTL headroom"
                    onFilter={() =>
                        deps.setColumnFilter("ttl_risk", {
                            op: AddressHistoryFilterOperator.IN,
                            values: [row.ttl_risk],
                        })
                    }
                >
                    {renderTtlRiskCell(row.ttl_risk)}
                </FilterableCell>
            ),
        },
    ];

    const hidden = deps.lockedFilterKey ? (HIDDEN_COLUMNS_WHEN_LOCKED[deps.lockedFilterKey] ?? []) : [];
    return allColumns.filter((c) => !hidden.includes(String(c.accessor)));
}
