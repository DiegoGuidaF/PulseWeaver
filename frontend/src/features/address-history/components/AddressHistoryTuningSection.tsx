import { useNavigate } from "react-router";
import { buildRoute } from "@/lib/routes";
import { AddressHistoryFilterOperator } from "@/lib/api";
import { DATEPICKER_TIME_PRESETS } from "@/lib/timePresets";
import { useDeviceRefs } from "@/features/devices/hooks/useDeviceRefs";
import { useAddressHistoryTuning } from "../hooks/useAddressHistoryTuning";
import { AddressHistoryTuningStrip } from "./AddressHistoryTuningStrip";
import type { AddressHistoryFilters } from "../hooks/useAddressHistoryFilters";

const PRESET_LABELS: Record<string, string> = Object.fromEntries(
    DATEPICKER_TIME_PRESETS.map((p) => [p.key, p.label]),
);

interface AddressHistoryTuningSectionProps {
    filters: AddressHistoryFilters;
    refreshInterval: number;
}

/**
 * Owns the tuning query and the navigation out of it. Deliberately reads
 * `windowParams` rather than `queryParams`: the recommendation is a fleet-level
 * reading over the time window, so it must not move when the events table below
 * is filtered. The chart there does follow those filters — it is that table's
 * own histogram — which is why this renders as its own section.
 */
export function AddressHistoryTuningSection({ filters, refreshInterval }: AddressHistoryTuningSectionProps) {
    const navigate = useNavigate();
    const { data: deviceRefs } = useDeviceRefs();

    const lockedDeviceId =
        filters.lockedFilter?.key === "device_id" ? Number(filters.lockedFilter.values[0]) : undefined;

    const { data, isPending, error, refetch } = useAddressHistoryTuning(
        { ...filters.windowParams, device_id: lockedDeviceId },
        refreshInterval === 0 ? false : refreshInterval,
    );

    /** Opens the device's auto-expiry control with the suggested TTL staged. */
    function goToDeviceTtl(deviceId: number, ttlSeconds: number) {
        const ownerId = (deviceRefs ?? []).find((d) => d.id === deviceId)?.owner_id;
        if (ownerId !== undefined) navigate(buildRoute.deviceRules(ownerId, deviceId, ttlSeconds));
    }

    function selectDevice(deviceId: number) {
        filters.setColumnFilter("device_id", {
            op: AddressHistoryFilterOperator.IN,
            values: [String(deviceId)],
        });
    }

    const windowLabel = filters.presetStr
        ? `last ${PRESET_LABELS[filters.presetStr] ?? filters.presetStr}`
        : "selected window";
    const deviceScoped = lockedDeviceId !== undefined;

    return (
        <AddressHistoryTuningStrip
            devices={data?.devices ?? []}
            total={data?.total ?? 0}
            minRenewals={data?.min_renewals ?? 0}
            scopeLabel={`${deviceScoped ? "this device" : "all devices"} · ${windowLabel}`}
            onSelectDevice={selectDevice}
            onTuneDevice={goToDeviceTtl}
            deviceScoped={deviceScoped}
            isPending={isPending}
            error={error}
            onRetry={() => refetch()}
        />
    );
}
