import { useEffect, useMemo, useReducer } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { ROUTES } from "@/lib/routes";
import { Alert, Center, Divider, Grid, Loader, Stack } from "@mantine/core";
import { IconAlertCircle } from "@tabler/icons-react";
import { notifications } from "@mantine/notifications";
import { useNetworkPolicy } from "@/features/network-policies/hooks/useNetworkPolicy";
import { useUpdateNetworkPolicy } from "@/features/network-policies/hooks/useUpdateNetworkPolicy";
import { useDeleteNetworkPolicy } from "@/features/network-policies/hooks/useDeleteNetworkPolicy";
import { useUpdateNetworkPolicyAccess } from "@/features/network-policies/hooks/useUpdateNetworkPolicyAccess";
import { NetworkPolicyHeader } from "@/features/network-policies/components/NetworkPolicyHeader";
import { SubjectGroupsPanel } from "@/features/subjects/components/SubjectGroupsPanel";
import { EffectiveHostsPanel } from "@/features/subjects/components/EffectiveHostsPanel";
import {
  subjectAccessReducer,
  initialSubjectAccessDraft,
  initDraftFromGroups,
  isSubjectAccessDirty,
} from "@/features/subjects/drafts/subjectAccessDraft";
import { buildModifyAccessRequest } from "@/features/subjects/drafts/saveSubjectAccessDraft";
import { useUnsavedChangesGuard } from "@/hooks/useUnsavedChangesGuard";
import { STAGED_BAR_HEIGHT } from "@/features/host-access/components/StagedChangesBar";
import { StagedChangesBar } from "@/features/host-access/components/StagedChangesBar";
import { toErrorMessage } from "@/lib/api-client";
import type { ModifyNetworkPolicyRequest } from "@/lib/api";

export function NetworkPolicyDetailPage() {
    const { id } = useParams();
    const navigate = useNavigate();
    const policyId = Number(id);

    const { data, isPending, isError } = useNetworkPolicy(policyId);
    const updateMutation = useUpdateNetworkPolicy();
    const deleteMutation = useDeleteNetworkPolicy();
    const accessMutation = useUpdateNetworkPolicyAccess();

    const [draft, dispatch] = useReducer(
        subjectAccessReducer,
        undefined,
        initialSubjectAccessDraft,
    );

    // Reset draft only when server grant state actually changes (not on background refetches)
    useEffect(() => {
        if (data) dispatch({ type: "reset", groups: data.groups, bypassHostCheck: data.bypass_host_check });
    // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [data?.updated_at]);

    const savedDraft = useMemo(
        () => (data ? initDraftFromGroups(data.groups, data.bypass_host_check) : null),
        [data],
    );
    const dirty = savedDraft != null && isSubjectAccessDirty(draft, savedDraft);

    useUnsavedChangesGuard(dirty);

    function handleUpdate(partial: Partial<ModifyNetworkPolicyRequest>) {
        if (!data) return;
        updateMutation.mutate(
            {
                path: { policy_id: policyId },
                body: {
                    name: data.name,
                    cidr: data.cidr,
                    description: data.description ?? "",
                    enabled: data.enabled,
                    ...partial,
                },
            },
            {
                onError: (err) =>
                    notifications.show({ color: "red", message: toErrorMessage(err) }),
            },
        );
    }

    function handleDelete() {
        deleteMutation.mutate(
            { path: { policy_id: policyId } },
            {
                onSuccess: () => navigate(ROUTES.accessNetworkPolicies),
                onError: (err) =>
                    notifications.show({ color: "red", message: toErrorMessage(err) }),
            },
        );
    }

    function handleSaveAccess() {
        accessMutation.mutate(
            { path: { policy_id: policyId }, body: buildModifyAccessRequest(draft) },
            {
                onError: (err) =>
                    notifications.show({ color: "red", message: toErrorMessage(err) }),
            },
        );
    }

    function handleDiscardAccess() {
        if (data) dispatch({ type: "reset", groups: data.groups, bypassHostCheck: data.bypass_host_check });
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

            <Grid>
                <Grid.Col span={{ base: 12, md: 5 }}>
                    <SubjectGroupsPanel
                        groups={data.groups}
                        draft={draft}
                        dispatch={dispatch}
                        disabled={accessMutation.isPending}
                    />
                </Grid.Col>
                <Grid.Col span={{ base: 12, md: 7 }}>
                    <EffectiveHostsPanel
                        groups={data.groups}
                        assignedGroupIds={draft.assignedGroupIds}
                        bypassHostCheck={draft.bypassHostCheck}
                    />
                </Grid.Col>
            </Grid>

            <StagedChangesBar
                visible={dirty}
                summary="You have unsaved access changes."
                saving={accessMutation.isPending}
                onSave={handleSaveAccess}
                onDiscard={handleDiscardAccess}
            />
        </Stack>
    );
}
