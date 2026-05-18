import { useState } from "react";
import { useNavigate } from "react-router-dom";
import {
    ActionIcon,
    Badge,
    Button,
    Center,
    Group,
    Progress,
    Stack,
    Switch,
    Text,
    Tooltip,
} from "@mantine/core";
import { IconNetwork, IconTrash } from "@tabler/icons-react";
import { DataTable } from "mantine-datatable";
import { notifications } from "@mantine/notifications";
import type { NetworkPolicySummary } from "@/lib/api";
import { useDateFormatter } from "@/contexts/useDateTimePrefs";
import { toErrorMessage } from "@/lib/api-client";
import { useUpdateNetworkPolicy } from "../hooks/useUpdateNetworkPolicy";
import { useDeleteNetworkPolicy } from "../hooks/useDeleteNetworkPolicy";
import { networkPolicyHostAccessSummary } from "../constants";
import { DeleteNetworkPolicyModal } from "./DeleteNetworkPolicyModal";

function HostAccessCell({ policy }: { policy: NetworkPolicySummary }) {
    const summary = networkPolicyHostAccessSummary(policy);
    if (summary === "—") {
        return (
            <Tooltip label="No host access configured — traffic is denied" withArrow>
                <Text size="sm" c="dimmed" style={{ cursor: "default" }}>
                    —
                </Text>
            </Tooltip>
        );
    }
    if (policy.allow_all_hosts) {
        return <Text size="sm">{summary}</Text>;
    }
    const pct = policy.total_host_count > 0
        ? (policy.effective_host_count / policy.total_host_count) * 100
        : 0;
    return (
        <Group gap="xs" wrap="nowrap" style={{ minWidth: 90 }}>
            <Text size="sm" style={{ whiteSpace: "nowrap" }}>{summary}</Text>
            <Progress value={pct} size="xs" style={{ flex: 1, minWidth: 40 }} color="indigo" />
        </Group>
    );
}

function EnableToggle({ policy }: { policy: NetworkPolicySummary }) {
    const updateMutation = useUpdateNetworkPolicy();

    return (
        <Switch
            size="xs"
            checked={policy.enabled}
            label={
                <Badge
                    variant="dot"
                    color={policy.enabled ? "green" : "gray"}
                    size="sm"
                >
                    {policy.enabled ? "On" : "Off"}
                </Badge>
            }
            onChange={(e) => {
                updateMutation.mutate(
                    { path: { id: policy.id }, body: { enabled: e.currentTarget.checked } },
                    {
                        onError: (err) =>
                            notifications.show({ color: "red", message: toErrorMessage(err) }),
                    },
                );
            }}
            onClick={(e) => e.stopPropagation()}
            disabled={updateMutation.isPending}
        />
    );
}

interface Props {
    policies: NetworkPolicySummary[];
    onNewPolicy: () => void;
}

export function NetworkPoliciesTable({ policies, onNewPolicy }: Props) {
    const navigate = useNavigate();
    const formatDateTime = useDateFormatter();
    const [deleteTarget, setDeleteTarget] = useState<NetworkPolicySummary | null>(null);

    const deleteMutation = useDeleteNetworkPolicy({ onSuccess: () => setDeleteTarget(null) });

    if (policies.length === 0) {
        return (
            <Center py="xl">
                <Stack align="center" gap="sm">
                    <IconNetwork size={40} color="var(--mantine-color-dimmed)" />
                    <Text fw={500}>No network policies configured.</Text>
                    <Text size="sm" c="dimmed" ta="center" maw={380}>
                        Add a policy to allow traffic from an IP range without registering
                        individual device addresses.
                    </Text>
                    <Button onClick={onNewPolicy}>+ New policy</Button>
                </Stack>
            </Center>
        );
    }

    return (
        <>
            <DataTable
                records={policies}
                highlightOnHover
                onRowClick={({ record }) => navigate(`/network-policies/${record.id}`)}
                columns={[
                    {
                        accessor: "name",
                        title: "Name",
                        render: (p) => <Text size="sm" fw={500}>{p.name}</Text>,
                    },
                    {
                        accessor: "cidr",
                        title: "CIDR",
                        render: (p) => <Text size="sm" ff="monospace">{p.cidr}</Text>,
                    },
                    {
                        accessor: "enabled",
                        title: "Status",
                        render: (p) => (
                            <EnableToggle policy={p} />
                        ),
                    },
                    {
                        accessor: "host_access",
                        title: "Host access",
                        render: (p) => <HostAccessCell policy={p} />,
                    },
                    {
                        accessor: "created_at",
                        title: "Created",
                        render: (p) => <Text size="sm" c="dimmed">{formatDateTime(p.created_at)}</Text>,
                    },
                    {
                        accessor: "actions",
                        title: "",
                        width: 48,
                        render: (p) => (
                            <ActionIcon
                                variant="subtle"
                                color="red"
                                size="sm"
                                aria-label="Delete policy"
                                onClick={(e) => { e.stopPropagation(); setDeleteTarget(p); }}
                            >
                                <IconTrash size={14} />
                            </ActionIcon>
                        ),
                    },
                ]}
            />

            <DeleteNetworkPolicyModal
                policyName={deleteTarget?.name ?? ""}
                opened={deleteTarget != null}
                isDeleting={deleteMutation.isPending}
                onClose={() => setDeleteTarget(null)}
                onConfirm={() => {
                    if (!deleteTarget) return;
                    deleteMutation.mutate(
                        { path: { id: deleteTarget.id } },
                        {
                            onError: (err) =>
                                notifications.show({ color: "red", message: toErrorMessage(err) }),
                        },
                    );
                }}
            />
        </>
    );
}
