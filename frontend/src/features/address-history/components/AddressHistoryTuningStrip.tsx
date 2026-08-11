import { Badge, Button, Card, Group, Stack, Text, UnstyledButton } from "@mantine/core";
import { IconAdjustments } from "@tabler/icons-react";
import type { AddressHistoryAtRiskDevice } from "@/lib/api";
import { formatTtlLabel, suggestTtlSeconds } from "@/lib/ttlPresets";
import { TTL_RISK_BADGE, TTL_RISK_LABELS, formatGapDuration } from "../constants";

interface AddressHistoryTuningStripProps {
    devices: AddressHistoryAtRiskDevice[];
    /** Applies the device filter. Omitted when the view is already device-scoped. */
    onSelectDevice?: (deviceId: number) => void;
    /** Opens the device's auto-expiry control, where the TTL is actually changed. */
    onTuneDevice: (deviceId: number) => void;
    deviceScoped?: boolean;
}

/**
 * The devices whose renewals crowd their lease TTL, with the numbers needed to
 * resize it: how often they renew, how often that lands late, and the gap the
 * TTL would have to cover. A ranking of worst offenders answers "who is
 * broken"; this answers "what should the TTL be", which is the only action this
 * screen can actually offer.
 */
export function AddressHistoryTuningStrip({
    devices,
    onSelectDevice,
    onTuneDevice,
    deviceScoped = false,
}: AddressHistoryTuningStripProps) {
    if (devices.length === 0) return null;

    return (
        <Card withBorder padding="md">
            <Text fw={500} mb="xs">
                {deviceScoped ? "TTL tuning" : "Devices worth tuning"}
            </Text>
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
        </Card>
    );
}

function TuningRow({
    device,
    onSelectDevice,
    onTuneDevice,
}: {
    device: AddressHistoryAtRiskDevice;
    onSelectDevice?: (deviceId: number) => void;
    onTuneDevice: (deviceId: number) => void;
}) {
    const badge = TTL_RISK_BADGE[device.worst_risk];
    const latePercent =
        device.renewal_count > 0 ? Math.round((device.late_renewal_count / device.renewal_count) * 100) : 0;

    const suggested = suggestTtlSeconds(device.p95_gap_seconds);
    // A suggestion at or below the current TTL is no suggestion: the device
    // already has enough headroom for 95% of its renewals, and the tail that
    // spills over is not worth a bigger lease.
    const showSuggestion = suggested === null || suggested > device.ttl_seconds;

    return (
        <Group gap="xs" wrap="wrap" justify="space-between">
            <Group gap="xs" wrap="wrap" style={{ flex: 1, minWidth: 240 }}>
                <Badge size="sm" color={badge.color} variant={badge.variant}>
                    {TTL_RISK_LABELS[device.worst_risk]}
                </Badge>
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
            {showSuggestion && (
                <Button
                    size="compact-xs"
                    variant="light"
                    leftSection={<IconAdjustments size={14} />}
                    onClick={() => onTuneDevice(device.device_id)}
                >
                    {suggested === null
                        ? "Set a custom TTL"
                        : `Set TTL ${formatTtlLabel(suggested)}`}
                </Button>
            )}
        </Group>
    );
}
