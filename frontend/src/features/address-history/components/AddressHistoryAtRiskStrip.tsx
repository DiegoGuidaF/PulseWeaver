import { Badge, Card, Group, Text, UnstyledButton } from "@mantine/core";
import { InfoTooltip } from "@/components/InfoTooltip";
import type { AddressHistoryAtRiskDevice } from "@/lib/api";
import { TTL_RISK_BADGE, TTL_RISK_LABELS } from "../constants";

interface AddressHistoryAtRiskStripProps {
    devices: AddressHistoryAtRiskDevice[];
    onSelectDevice: (deviceId: number) => void;
}

/**
 * Ranking of the devices most at risk of losing access in the filtered
 * window — the answer a time series structurally cannot give. Absent
 * entirely on the device-scoped view, where the ranking degenerates to the
 * one device already being viewed.
 */
export function AddressHistoryAtRiskStrip({ devices, onSelectDevice }: AddressHistoryAtRiskStripProps) {
    if (devices.length === 0) return null;

    return (
        <Card withBorder padding="md">
            <Group gap={6} mb="xs" wrap="nowrap">
                <Text fw={500}>At-risk devices</Text>
                <InfoTooltip label="The number beside each device is its total events in this window across every risk level — not a count of breaches." />
            </Group>
            <Group gap="lg" wrap="wrap">
                {devices.map((d) => {
                    const badge = TTL_RISK_BADGE[d.worst_risk];
                    return (
                        <UnstyledButton
                            key={d.device_id}
                            onClick={() => onSelectDevice(d.device_id)}
                            aria-label={`Filter by ${d.device_name}`}
                            style={{ display: "inline-flex", alignItems: "center", minHeight: 24 }}
                        >
                            <Group gap={6} wrap="nowrap">
                                <Badge size="sm" color={badge.color} variant={badge.variant}>
                                    {TTL_RISK_LABELS[d.worst_risk]}
                                </Badge>
                                <Text size="sm" fw={500} c="indigo" td="underline">
                                    {d.device_name}
                                </Text>
                                <Text size="xs" c="dimmed">
                                    {d.event_count} event{d.event_count === 1 ? "" : "s"}
                                </Text>
                            </Group>
                        </UnstyledButton>
                    );
                })}
            </Group>
        </Card>
    );
}
