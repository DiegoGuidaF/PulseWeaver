import { Badge, Button, Card, Group, Skeleton, Stack, Text, UnstyledButton } from "@mantine/core";
import { IconAdjustments } from "@tabler/icons-react";
import type { AddressHistoryTuningCandidate } from "@/lib/api";
import { ErrorState } from "@/components/ErrorState";
import { formatTtlLabel, suggestTtlSeconds } from "@/lib/ttlPresets";
import { formatGapDuration } from "../constants";

interface AddressHistoryTuningStripProps {
    devices: AddressHistoryTuningCandidate[];
    /** Devices meeting the threshold fleet-wide; `devices` is the worst few of it. */
    total: number;
    /** Sample floor the API applied, echoed so the disclosure states the real number. */
    minRenewals: number;
    /** What the readout was computed over, e.g. "all devices · last 1w". */
    scopeLabel: string;
    /** Applies the device filter. Omitted when the view is already device-scoped. */
    onSelectDevice?: (deviceId: number) => void;
    /**
     * Opens the device's auto-expiry control with `ttlSeconds` staged, where the
     * user confirms it. The strip never writes a TTL itself.
     */
    onTuneDevice: (deviceId: number, ttlSeconds: number) => void;
    deviceScoped?: boolean;
    isPending: boolean;
    error?: unknown;
    onRetry?: () => void;
}

/**
 * The devices whose renewals outgrew their lease TTL, with the numbers needed to
 * resize it: how often they renew, how often that lands late, and the gap the
 * TTL would have to cover. A ranking of worst offenders answers "who is
 * broken"; this answers "what should the TTL be", which is the only action this
 * screen can actually offer.
 *
 * Scoped to the time window alone. The events table below filters independently,
 * so a fleet-level recommendation does not shift as the reader slices rows.
 */
export function AddressHistoryTuningStrip({
    devices,
    total,
    minRenewals,
    scopeLabel,
    onSelectDevice,
    onTuneDevice,
    deviceScoped = false,
    isPending,
    error,
    onRetry,
}: AddressHistoryTuningStripProps) {
    return (
        <Card withBorder padding="md">
            <Group justify="space-between" align="baseline" wrap="wrap" mb="xs">
                <Text fw={500}>{deviceScoped ? "TTL tuning" : "Devices worth tuning"}</Text>
                <Text size="xs" c="dimmed">
                    {scopeLabel}
                </Text>
            </Group>

            {isPending ? (
                <Skeleton height={72} radius="sm" />
            ) : error ? (
                <ErrorState error={error} title="Failed to load TTL tuning" onRetry={onRetry} />
            ) : devices.length === 0 ? (
                // Absence of a recommendation is a reading in its own right, so it
                // gets a line rather than collapsing the section to nothing.
                <Text size="sm" c="dimmed">
                    {deviceScoped
                        ? "This device's TTL comfortably covers its renewals in this window."
                        : "No device's TTL needs resizing in this window."}
                </Text>
            ) : (
                <Stack gap="xs">
                    {devices.map((d) => (
                        <TuningRow
                            key={d.device_id}
                            device={d}
                            onSelectDevice={deviceScoped ? undefined : onSelectDevice}
                            onTuneDevice={onTuneDevice}
                        />
                    ))}
                </Stack>
            )}

            {!isPending && !error && !deviceScoped && (
                <Text size="xs" c="dimmed" mt="sm">
                    {total > devices.length
                        ? `Showing the ${devices.length} widest misses of ${total} devices. `
                        : ""}
                    Listing devices with at least {minRenewals} renewals whose TTL misses more than 5% of
                    them.
                </Text>
            )}
        </Card>
    );
}

function TuningRow({
    device,
    onSelectDevice,
    onTuneDevice,
}: {
    device: AddressHistoryTuningCandidate;
    onSelectDevice?: (deviceId: number) => void;
    onTuneDevice: (deviceId: number, ttlSeconds: number) => void;
}) {
    const latePercent =
        device.renewal_count > 0 ? Math.round((device.late_renewal_count / device.renewal_count) * 100) : 0;

    const suggested = suggestTtlSeconds(device.p95_gap_seconds);
    // Device-scoped, the readout also covers devices that pass the threshold, so
    // a TTL already wide enough gets a reading and no call to action.
    const needsTuning = device.p95_gap_seconds > device.ttl_seconds;

    return (
        <Group gap="xs" wrap="wrap" justify="space-between">
            <Group gap="xs" wrap="wrap" style={{ flex: 1, minWidth: 240 }}>
                {needsTuning ? (
                    // The size of the miss, not the fact of it: every listed
                    // device is past its TTL by construction, so a "past TTL"
                    // badge here would be a constant.
                    <Badge size="sm" color="orange" variant="light">
                        {formatTtlLabel(device.ttl_seconds)} → {suggested === null ? "custom" : formatTtlLabel(suggested)}
                    </Badge>
                ) : (
                    <Badge size="sm" color="indigo" variant="light">
                        Comfortable
                    </Badge>
                )}
                {onSelectDevice ? (
                    <UnstyledButton
                        onClick={() => onSelectDevice(device.device_id)}
                        aria-label={`Filter by ${device.device_name}`}
                        style={{ display: "inline-flex", alignItems: "center", minHeight: 24 }}
                    >
                        <Text size="sm" fw={500} c="indigo" td="underline">
                            {device.device_name}
                        </Text>
                    </UnstyledButton>
                ) : (
                    <Text size="sm" fw={500}>
                        {device.device_name}
                    </Text>
                )}
                <Text size="xs" c="dimmed">
                    TTL {formatTtlLabel(device.ttl_seconds)} · {device.renewal_count} renewal
                    {device.renewal_count === 1 ? "" : "s"} · {device.late_renewal_count} past TTL ({latePercent}%) ·
                    p95 gap {formatGapDuration(device.p95_gap_seconds)}
                </Text>
            </Group>
            {needsTuning && (
                <Button
                    size="compact-xs"
                    variant="light"
                    leftSection={<IconAdjustments size={14} />}
                    // A gap past the top of the ladder has no preset to offer, so
                    // it travels raw and the control falls back to its custom
                    // input — one staging path behind both button labels.
                    onClick={() => onTuneDevice(device.device_id, suggested ?? device.p95_gap_seconds)}
                >
                    {suggested === null ? "Set a custom TTL" : `Set TTL ${formatTtlLabel(suggested)}`}
                </Button>
            )}
        </Group>
    );
}
