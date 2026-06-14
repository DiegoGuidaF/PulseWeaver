import { Stack, SimpleGrid, Group, Text, Tooltip, ThemeIcon } from "@mantine/core";
import { IconInfoCircle } from "@tabler/icons-react";
import { useAttributionSplit } from "../hooks/useAttributionSplit";
import { AttributionTable } from "./AttributionTable";

interface AttributionSectionProps {
    from?: string;
    to?: string;
}

const CAVEAT =
    "A single request can match several entities (shared IPs, multiple devices), and each match is counted. These are per-entity shares, not a partition of total traffic — the columns can sum above the totals above.";

export function AttributionSection({ from, to }: AttributionSectionProps) {
    const policy = useAttributionSplit("policy", from, to);
    const user = useAttributionSplit("user", from, to);
    const device = useAttributionSplit("device", from, to);

    return (
        <Stack gap="xs">
            <Group gap={6} align="center">
                <Text fw={500}>Traffic by entity</Text>
                <Tooltip label={CAVEAT} multiline w={280} withArrow position="right">
                    <ThemeIcon variant="transparent" color="gray" size="sm" aria-label="How per-entity traffic is counted">
                        <IconInfoCircle size={16} stroke={1.5} />
                    </ThemeIcon>
                </Tooltip>
            </Group>
            <Text size="xs" c="dimmed">
                Per-entity shares — a shared request is counted under each matching entity, so these do not sum to total
                traffic.
            </Text>

            <SimpleGrid cols={{ base: 1, lg: 3 }}>
                <AttributionTable
                    title="By network policy"
                    entityHeader="Policy"
                    data={policy.data?.entities}
                    isLoading={policy.isLoading}
                    error={policy.error}
                    onRetry={() => policy.refetch()}
                    emptyText="No policy-matched traffic in this period"
                />
                <AttributionTable
                    title="By user"
                    entityHeader="User"
                    data={user.data?.entities}
                    isLoading={user.isLoading}
                    error={user.error}
                    onRetry={() => user.refetch()}
                    emptyText="No user-attributed traffic in this period"
                />
                <AttributionTable
                    title="By device"
                    entityHeader="Device"
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
