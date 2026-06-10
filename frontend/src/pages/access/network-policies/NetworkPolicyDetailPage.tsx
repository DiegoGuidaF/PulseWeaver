import { useEffect, useMemo, useReducer } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { ROUTES } from "@/lib/routes";
import { Center, Divider, Grid, Loader, Stack } from "@mantine/core";
import { notifications } from "@mantine/notifications";
import { ErrorState } from "@/components/ErrorState";
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
  isBypassJustEnabled,
  requiresBypassAcknowledgement,
} from "@/features/subjects/drafts/subjectAccessDraft";
import { buildModifyAccessRequest } from "@/features/subjects/drafts/saveSubjectAccessDraft";
import { useUnsavedChangesGuard } from "@/hooks/useUnsavedChangesGuard";
import { StagedChangesBar, STAGED_BAR_HEIGHT } from "@/features/host-access/components/StagedChangesBar";
import { formatAddressCount } from "@/features/network-policies/constants";
import { toErrorMessage } from "@/lib/api-client";
import type { ModifyNetworkPolicyRequest } from "@/lib/api";

export function NetworkPolicyDetailPage() {
    const { id } = useParams();
    const navigate = useNavigate();
    const policyId = Number(id);

    const { data, isPending, isError, error, refetch } = useNetworkPolicy(policyId);
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

    // Bypass grants the whole CIDR access to every host (including future
    // ones), warn about it and ensure user confirms
    const bypassJustEnabled = savedDraft != null && isBypassJustEnabled(savedDraft, draft);
    const bypassAckRequired =
        savedDraft != null && requiresBypassAcknowledgement(savedDraft, draft);

    useUnsavedChangesGuard(dirty);

    function handleUpdate(
        partial: Partial<ModifyNetworkPolicyRequest>,
        opts?: { onSuccess?: () => void },
    ) {
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
                onSuccess: () => opts?.onSuccess?.(),
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
        if (bypassAckRequired) return;
        accessMutation.mutate(
            { path: { policy_id: policyId }, body: buildModifyAccessRequest(draft) },
            {
                onSuccess: () => notifications.show({ color: "green", message: "Access updated" }),
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
            <ErrorState
                error={error}
                title="Could not load policy"
                message={error ? undefined : "This network policy could not be loaded."}
                onRetry={() => refetch()}
            />
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
                summary={
                    bypassJustEnabled
                        ? "You're about to enable host-check bypass."
                        : "You have unsaved access changes."
                }
                saving={accessMutation.isPending}
                onSave={handleSaveAccess}
                onDiscard={handleDiscardAccess}
                warning={
                    bypassJustEnabled
                        ? {
                              detail: `Enabling bypass lets ${formatAddressCount(data.cidr) ?? "every address"} in ${data.cidr} reach all hosts, including future ones.`,
                              acknowledgeLabel: "I understand this exposes every host in this range to the whole CIDR.",
                              acknowledged: draft.bypassAcknowledged,
                              onAcknowledgeChange: (value) => dispatch({ type: "acknowledgeBypass", value }),
                          }
                        : undefined
                }
            />
        </Stack>
    );
}
