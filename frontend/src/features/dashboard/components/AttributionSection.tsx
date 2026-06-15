import { Stack, SimpleGrid, Text } from "@mantine/core";
import { buildRoute } from "@/lib/routes";
import { useAttributionSplit } from "../hooks/useAttributionSplit";
import { AttributionTable } from "./AttributionTable";

interface AttributionSectionProps {
    from?: string;
    to?: string;
}

export function AttributionSection({ from, to }: AttributionSectionProps) {
    const policy = useAttributionSplit("policy", from, to);
    const user = useAttributionSplit("user", from, to);
    const device = useAttributionSplit("device", from, to);

    return (
        <Stack gap="xs">
            <Text fw={500}>Traffic by entity</Text>
            <Text size="xs" c="dimmed">
                Per-entity shares — a shared request is counted under each matching entity, so these do not sum to total
                traffic.
            </Text>

            <SimpleGrid cols={{ base: 1, lg: 3 }}>
                <AttributionTable
                    title="By network policy"
                    entityHeader="Policy"
                    entityHeaderPlural="Policies"
                    data={policy.data?.entities}
                    isLoading={policy.isLoading}
                    error={policy.error}
                    onRetry={() => policy.refetch()}
                    emptyText="No policy-matched traffic in this period"
                    rowHref={(row) =>
                        row.entity_id != null ? buildRoute.accessNetworkPolicyDetail(row.entity_id) : undefined
                    }
                />
                <AttributionTable
                    title="By user"
                    entityHeader="User"
                    entityHeaderPlural="Users"
                    data={user.data?.entities}
                    isLoading={user.isLoading}
                    error={user.error}
                    onRetry={() => user.refetch()}
                    emptyText="No user-attributed traffic in this period"
                    rowHref={(row) => (row.entity_id != null ? buildRoute.accessUserDetail(row.entity_id) : undefined)}
                />
                {/* Devices have no detail route reachable from this payload (it carries the device id, not its
                    owner), so device rows are not linkable until the attribution endpoint returns owner_id. */}
                <AttributionTable
                    title="By device"
                    entityHeader="Device"
                    entityHeaderPlural="Devices"
                    data={device.data?.entities}
                    isLoading={device.isLoading}
                    error={device.error}
                    onRetry={() => device.refetch()}
                    emptyText="No device-attributed traffic in this period"
                />
            </SimpleGrid>
        </Stack>
    );
}
