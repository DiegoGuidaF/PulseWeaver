import { Anchor, Group, Skeleton, Stack, Text } from "@mantine/core";
import { Link } from "react-router-dom";
import { notifications } from "@mantine/notifications";
import { ErrorState } from "@/components/ErrorState";
import { ROUTES } from "@/lib/routes";
import { toErrorMessage } from "@/lib/api-client";
import type { Anomaly } from "@/lib/api";
import { useOpenAnomalies } from "../hooks/useOpenAnomalies";
import { useAcknowledgeAnomaly } from "../hooks/useAcknowledgeAnomaly";
import { DASHBOARD_ANOMALY_ROW_LIMIT, DASHBOARD_ANOMALY_SEVERITIES } from "../constants";
import { AnomalyRow } from "./AnomalyRow";

/**
 * Dashboard "Unusual activity" section — open anomalies at warning/critical
 * severity, capped and linking out to the full Anomalies page. This is
 * "now" state like the posture strip, not scoped by the traffic time-range
 * preset, so it sits outside the Traffic section in `DashboardView`.
 */
export function AnomalySection() {
    const { data, isPending, isError, refetch } = useOpenAnomalies();
    const acknowledge = useAcknowledgeAnomaly();

    const handleAcknowledge = (anomaly: Anomaly) => {
        acknowledge.mutate(
            { path: { id: anomaly.id } },
            {
                onError: (err) =>
                    notifications.show({
                        color: "red",
                        title: "Couldn't acknowledge anomaly",
                        message: toErrorMessage(err),
                    }),
            },
        );
    };

    if (isPending) {
        return (
            <Stack gap="sm">
                <Text fw={600}>Unusual activity</Text>
                <Skeleton h={72} radius="md" />
            </Stack>
        );
    }

    if (isError) {
        return (
            <Stack gap="sm">
                <Text fw={600}>Unusual activity</Text>
                <ErrorState title="Failed to load anomalies" onRetry={() => refetch()} />
            </Stack>
        );
    }

    const allOpen = data?.anomalies ?? [];
    const visible = allOpen
        .filter((a) => DASHBOARD_ANOMALY_SEVERITIES.has(a.severity))
        .slice(0, DASHBOARD_ANOMALY_ROW_LIMIT);
    const totalOpenCount = allOpen.length;

    return (
        <Stack gap="sm">
            <Group justify="space-between" align="center" wrap="wrap">
                <Text fw={600}>Unusual activity</Text>
                {totalOpenCount > visible.length && (
                    <Anchor component={Link} to={ROUTES.anomalies} size="sm">
                        View all {totalOpenCount} →
                    </Anchor>
                )}
            </Group>

            {visible.length === 0 ? (
                <Text size="sm" c="dimmed">
                    No unusual activity
                </Text>
            ) : (
                <Stack gap="xs">
                    {visible.map((anomaly) => (
                        <AnomalyRow
                            key={anomaly.id}
                            anomaly={anomaly}
                            dateDisplay="relative"
                            onAcknowledge={handleAcknowledge}
                            isAcknowledging={acknowledge.isPending && acknowledge.variables?.path?.id === anomaly.id}
                        />
                    ))}
                </Stack>
            )}
        </Stack>
    );
}
