import { useState } from "react";
import { Skeleton, Stack } from "@mantine/core";
import { notifications } from "@mantine/notifications";
import { IconMoodEmpty } from "@tabler/icons-react";
import { EmptyState } from "@/components/EmptyState";
import { ErrorState } from "@/components/ErrorState";
import { toErrorMessage } from "@/lib/api-client";
import { AnomalyStatus, type Anomaly } from "@/lib/api";
import { useAnomalies } from "../hooks/useAnomalies";
import { useAcknowledgeAnomaly } from "../hooks/useAcknowledgeAnomaly";
import { ANOMALIES_PAGE_LIMIT } from "../constants";
import { AnomaliesFilterBar, type AnomaliesFilterState } from "./AnomaliesFilterBar";
import { AnomalyRow } from "./AnomalyRow";

const DEFAULT_FILTERS: AnomaliesFilterState = {
    status: AnomalyStatus.OPEN,
    severity: null,
    kinds: [],
};

/**
 * Filter bar + result list for the Anomalies page. The default filter state
 * (`status: "open"`, no severity/kind) issues the exact same query as the
 * shared open-anomalies query (`useOpenAnomalies`), so landing on this page
 * reads the badge/dashboard's cache entry instead of triggering a refetch.
 */
export function AnomaliesList() {
    const [filters, setFilters] = useState<AnomaliesFilterState>(DEFAULT_FILTERS);
    const acknowledge = useAcknowledgeAnomaly();

    const { data, isPending, isError, refetch } = useAnomalies({
        status: filters.status === "all" ? undefined : filters.status,
        severity: filters.severity ?? undefined,
        kind: filters.kinds.length > 0 ? filters.kinds : undefined,
        limit: ANOMALIES_PAGE_LIMIT,
    });

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

    const rows = data?.anomalies ?? [];

    return (
        <Stack gap="md">
            <AnomaliesFilterBar filters={filters} onChange={setFilters} />

            {isPending ? (
                <Stack gap="xs">
                    {Array.from({ length: 4 }).map((_, i) => (
                        <Skeleton key={i} h={90} radius="md" />
                    ))}
                </Stack>
            ) : isError ? (
                <ErrorState title="Failed to load anomalies" onRetry={() => refetch()} />
            ) : rows.length === 0 ? (
                <EmptyState icon={IconMoodEmpty} title="No anomalies match these filters" />
            ) : (
                <Stack gap="xs">
                    {rows.map((anomaly) => (
                        <AnomalyRow
                            key={anomaly.id}
                            anomaly={anomaly}
                            expandable
                            dateDisplay="window"
                            onAcknowledge={handleAcknowledge}
                            isAcknowledging={acknowledge.isPending && acknowledge.variables?.path?.id === anomaly.id}
                        />
                    ))}
                </Stack>
            )}
        </Stack>
    );
}
