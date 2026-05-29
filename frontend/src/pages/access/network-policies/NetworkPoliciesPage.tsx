import { useState } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { buildRoute } from "@/lib/routes";
import { Button, Center, Group, Loader, Stack, Text, Title } from "@mantine/core";
import type { NetworkPolicyDetail } from "@/lib/api";
import { ErrorState } from "@/components/ErrorState";
import { useNetworkPolicies } from "@/features/network-policies/hooks/useNetworkPolicies";
import { NetworkPoliciesTable } from "@/features/network-policies/components/NetworkPoliciesTable";
import { CreateNetworkPolicyModal } from "@/features/network-policies/components/CreateNetworkPolicyModal";

export function NetworkPoliciesPage() {
    const navigate = useNavigate();
    const [searchParams] = useSearchParams();
    const [createOpen, setCreateOpen] = useState(false);
    const { data, isPending, isError, error, refetch } = useNetworkPolicies();

    const groupIdFilter = searchParams.get("group_id");
    const displayedPolicies = groupIdFilter && data
        ? data.filter((p) => p.groups.some((g) => g.id === Number(groupIdFilter)))
        : data;

    function handleCreated(policy: NetworkPolicyDetail) {
        setCreateOpen(false);
        navigate(buildRoute.accessNetworkPolicyDetail(policy.id));
    }

    return (
        <Stack maw={1200} gap="md">
            <Group justify="space-between" align="flex-start">
                <div>
                    <Title order={1}>Network Policies</Title>
                    <Text c="dimmed" mt={4}>
                        Configure named IP ranges that grant access independently of user devices.
                    </Text>
                </div>
                <Button onClick={() => setCreateOpen(true)}>+ New policy</Button>
            </Group>

            {isPending && (
                <Center py="xl">
                    <Loader />
                </Center>
            )}

            {isError && (
                <ErrorState
                    error={error}
                    title="Failed to load network policies"
                    onRetry={() => refetch()}
                />
            )}

            {displayedPolicies && (
                <NetworkPoliciesTable
                    policies={displayedPolicies}
                    onNewPolicy={() => setCreateOpen(true)}
                />
            )}

            <CreateNetworkPolicyModal
                opened={createOpen}
                onClose={() => setCreateOpen(false)}
                onCreated={handleCreated}
            />
        </Stack>
    );
}
