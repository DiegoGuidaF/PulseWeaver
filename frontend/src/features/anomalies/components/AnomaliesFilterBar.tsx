import { Group, MultiSelect, Select, SegmentedControl, Stack } from "@mantine/core";
import { AnomalyStatus, type AnomalyKind, type AnomalySeverity } from "@/lib/api";
import {
    ANOMALY_KIND_GROUPED_OPTIONS,
    ANOMALY_SEVERITY_OPTIONS,
    ANOMALY_STATUS_FILTER_OPTIONS,
    type AnomalyStatusFilter,
} from "../constants";

export interface AnomaliesFilterState {
    status: AnomalyStatusFilter;
    severity: AnomalySeverity | null;
    kinds: AnomalyKind[];
}

interface AnomaliesFilterBarProps {
    filters: AnomaliesFilterState;
    onChange: (filters: AnomaliesFilterState) => void;
}

const STATUS_DATA = ANOMALY_STATUS_FILTER_OPTIONS.map((o) => ({ value: o.value, label: o.label }));

export function AnomaliesFilterBar({ filters, onChange }: AnomaliesFilterBarProps) {
    return (
        <Stack gap="xs">
            <Group gap="sm" wrap="wrap">
                {/* A 3-option control fits fine at any width, but stay consistent with the
                    app's segmented-control-collapses-to-select convention below `sm`. */}
                <SegmentedControl
                    visibleFrom="sm"
                    value={filters.status}
                    onChange={(value) => onChange({ ...filters, status: value as AnomalyStatusFilter })}
                    data={STATUS_DATA}
                    aria-label="Filter by status"
                />
                <Select
                    hiddenFrom="sm"
                    style={{ flex: 1, minWidth: 140 }}
                    data={STATUS_DATA}
                    value={filters.status}
                    onChange={(value) =>
                        onChange({ ...filters, status: (value as AnomalyStatusFilter) ?? AnomalyStatus.OPEN })
                    }
                    allowDeselect={false}
                    aria-label="Filter by status"
                />
                <Select
                    placeholder="All severities"
                    clearable
                    data={ANOMALY_SEVERITY_OPTIONS}
                    value={filters.severity}
                    onChange={(value) => onChange({ ...filters, severity: value as AnomalySeverity | null })}
                    style={{ flex: 1, minWidth: 160 }}
                    aria-label="Filter by severity"
                />
            </Group>
            <MultiSelect
                placeholder="All kinds"
                clearable
                data={ANOMALY_KIND_GROUPED_OPTIONS}
                value={filters.kinds}
                onChange={(value) => onChange({ ...filters, kinds: value as AnomalyKind[] })}
                aria-label="Filter by kind"
            />
        </Stack>
    );
}
