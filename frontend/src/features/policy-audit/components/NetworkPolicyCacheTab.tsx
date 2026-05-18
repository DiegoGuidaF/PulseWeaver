import { useNavigate } from "react-router-dom";
import { Badge, Card, Group, Progress, Table, Text } from "@mantine/core";
import type { PolicyNetworkPolicyEntry } from "@/lib/api";

interface Props {
    entries: PolicyNetworkPolicyEntry[];
    totalHosts: number;
}

function EffectiveHostsCell({ entry, totalHosts }: { entry: PolicyNetworkPolicyEntry; totalHosts: number }) {
    if (entry.allow_all_hosts) {
        return <Text size="sm">All hosts</Text>;
    }
    const pct = totalHosts > 0 ? (entry.effective_host_count / totalHosts) * 100 : 0;
    return (
        <Group gap="xs" wrap="nowrap" style={{ minWidth: 90 }}>
            <Text size="sm" style={{ whiteSpace: "nowrap" }}>
                {entry.effective_host_count} / {totalHosts}
            </Text>
            <Progress value={pct} size="xs" style={{ flex: 1, minWidth: 40 }} color="indigo" />
        </Group>
    );
}

export function NetworkPolicyCacheTab({ entries, totalHosts }: Props) {
    const navigate = useNavigate();

    if (entries.length === 0) {
        return (
            <Text size="sm" c="dimmed" ta="center" py="xl">
                No network policies in the cache snapshot.
            </Text>
        );
    }

    return (
        <Card withBorder p={0}>
            <Table.ScrollContainer minWidth={600}>
                <Table highlightOnHover>
                    <Table.Thead>
                        <Table.Tr>
                            <Table.Th>Policy name</Table.Th>
                            <Table.Th>CIDR</Table.Th>
                            <Table.Th>Status</Table.Th>
                            <Table.Th>Effective hosts</Table.Th>
                        </Table.Tr>
                    </Table.Thead>
                    <Table.Tbody>
                        {entries.map((entry) => (
                            <Table.Tr
                                key={entry.policy_id}
                                style={{
                                    opacity: entry.enabled ? 1 : 0.5,
                                    cursor: "pointer",
                                }}
                                onClick={() => navigate(`/network-policies/${entry.policy_id}`)}
                            >
                                <Table.Td>
                                    <Text size="sm" fw={500}>{entry.policy_name}</Text>
                                </Table.Td>
                                <Table.Td>
                                    <Text size="sm" ff="monospace">{entry.cidr}</Text>
                                </Table.Td>
                                <Table.Td>
                                    <Badge
                                        variant="dot"
                                        color={entry.enabled ? "green" : "gray"}
                                        size="sm"
                                    >
                                        {entry.enabled ? "On" : "Off"}
                                    </Badge>
                                </Table.Td>
                                <Table.Td>
                                    <EffectiveHostsCell entry={entry} totalHosts={totalHosts} />
                                </Table.Td>
                            </Table.Tr>
                        ))}
                    </Table.Tbody>
                </Table>
            </Table.ScrollContainer>
        </Card>
    );
}
