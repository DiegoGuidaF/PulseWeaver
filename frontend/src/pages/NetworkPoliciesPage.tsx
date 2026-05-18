import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { Alert, Button, Center, Group, Loader, Stack, Text, Title } from "@mantine/core";
import { IconAlertCircle } from "@tabler/icons-react";
import type { NetworkPolicyDetail } from "@/lib/api";
import { useNetworkPolicies } from "@/features/network-policies/hooks/useNetworkPolicies";
import { NetworkPoliciesTable } from "@/features/network-policies/components/NetworkPoliciesTable";
import { CreateNetworkPolicyModal } from "@/features/network-policies/components/CreateNetworkPolicyModal";

export function NetworkPoliciesPage() {
    const navigate = useNavigate();
    const [createOpen, setCreateOpen] = useState(false);
    const { data, isPending, isError } = useNetworkPolicies();

    function handleCreated(policy: NetworkPolicyDetail) {
        setCreateOpen(false);
        navigate(`/access/network-policies/${policy.id}`);
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
                <Alert icon={<IconAlertCircle size={16} />} color="red" title="Failed to load">
                    Could not fetch network policies.
                </Alert>
            )}

            {data && (
                <NetworkPoliciesTable
                    policies={data}
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
