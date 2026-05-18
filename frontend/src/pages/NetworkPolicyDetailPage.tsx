import { useEffect, useMemo, useReducer } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { Alert, Card, Center, Divider, Loader, Stack, Text, Title } from "@mantine/core";
import { IconAlertCircle } from "@tabler/icons-react";
import { notifications } from "@mantine/notifications";
import { useNetworkPolicy } from "@/features/network-policies/hooks/useNetworkPolicy";
import { useUpdateNetworkPolicy } from "@/features/network-policies/hooks/useUpdateNetworkPolicy";
import { useDeleteNetworkPolicy } from "@/features/network-policies/hooks/useDeleteNetworkPolicy";
import { useUpdateNetworkPolicyHostAccess } from "@/features/network-policies/hooks/useUpdateNetworkPolicyHostAccess";
import { NetworkPolicyHeader } from "@/features/network-policies/components/NetworkPolicyHeader";
import { NetworkPolicyHostAccessPanel } from "@/features/network-policies/components/NetworkPolicyHostAccessPanel";
import {
    networkPolicyHostAccessReducer,
    initialNetworkPolicyHostAccessDraft,
    initDraftFromDetail,
    isHostAccessDirty,
} from "@/features/network-policies/drafts/networkPolicyHostAccessDraft";
import { buildHostAccessBody } from "@/features/network-policies/drafts/saveNetworkPolicyHostAccessDraft";
import { useUnsavedChangesGuard } from "@/hooks/useUnsavedChangesGuard";
import { STAGED_BAR_HEIGHT } from "@/features/host-access/components/StagedChangesBar";
import { toErrorMessage } from "@/lib/api-client";
import type { UpdateNetworkPolicyRequest } from "@/lib/api";

export function NetworkPolicyDetailPage() {
    const { id } = useParams();
    const navigate = useNavigate();
    const policyId = Number(id);

    const { data, isPending, isError } = useNetworkPolicy(policyId);
    const updateMutation = useUpdateNetworkPolicy();
    const deleteMutation = useDeleteNetworkPolicy();
    const hostAccessMutation = useUpdateNetworkPolicyHostAccess();

    const [draft, dispatch] = useReducer(
        networkPolicyHostAccessReducer,
        undefined,
        initialNetworkPolicyHostAccessDraft,
    );

    // Sync draft from server data keyed on updated_at — only resets when server state changes
    // (i.e., after a successful save triggers invalidation + refetch), not on background refetches.
    useEffect(() => {
        if (data) dispatch({ type: "reset", detail: data });
    // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [data?.updated_at]);

    const savedDraft = useMemo(() => (data ? initDraftFromDetail(data) : null), [data]);
    const dirty = savedDraft != null && isHostAccessDirty(draft, savedDraft);

    useUnsavedChangesGuard(dirty);

    function handleUpdate(fields: UpdateNetworkPolicyRequest) {
        updateMutation.mutate(
            { path: { id: policyId }, body: fields },
            {
                onError: (err) =>
                    notifications.show({ color: "red", message: toErrorMessage(err) }),
            },
        );
    }

    function handleDelete() {
        deleteMutation.mutate(
            { path: { id: policyId } },
            {
                onSuccess: () => navigate("/network-policies"),
                onError: (err) =>
                    notifications.show({ color: "red", message: toErrorMessage(err) }),
            },
        );
    }

    function handleSaveHostAccess() {
        if (!savedDraft) return;
        hostAccessMutation.mutate(
            { path: { id: policyId }, body: buildHostAccessBody(draft) },
            {
                onError: (err) =>
                    notifications.show({ color: "red", message: toErrorMessage(err) }),
            },
        );
    }

    function handleDiscardHostAccess() {
        if (data) dispatch({ type: "reset", detail: data });
    }

    if (isPending) {
        return (
            <Center py="xl">
                <Loader />
            </Center>
        );
    }

    if (isError || !data) {
        return (
            <Alert icon={<IconAlertCircle size={16} />} color="red" title="Not found">
                This network policy could not be loaded.
            </Alert>
        );
    }

    return (
        <Stack maw={1200} gap="lg" pb={dirty ? STAGED_BAR_HEIGHT : undefined}>
            <NetworkPolicyHeader
                policy={data}
                onUpdate={handleUpdate}
                onDelete={handleDelete}
                isUpdating={updateMutation.isPending}
                isDeleting={deleteMutation.isPending}
            />

            <Divider />

            <div>
                <Title order={4} mb="xs">
                    HOST ACCESS
                </Title>
                <Text size="sm" c="dimmed" mb="md">
                    Define which hosts are reachable for traffic matched by this policy.
                </Text>
                <Card withBorder p="lg">
                    <NetworkPolicyHostAccessPanel
                        detail={data}
                        draft={draft}
                        dispatch={dispatch}
                        isDirty={dirty}
                        isSaving={hostAccessMutation.isPending}
                        onSave={handleSaveHostAccess}
                        onDiscard={handleDiscardHostAccess}
                    />
                </Card>
            </div>
        </Stack>
    );
}
