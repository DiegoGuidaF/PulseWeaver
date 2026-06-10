import { useEffect, useMemo, useReducer, useState } from "react";
import { Link, useNavigate, useParams, useSearchParams } from "react-router-dom";
import { ROUTES } from "@/lib/routes";
import {
  Anchor,
  Badge,
  Button,
  Center,
  Grid,
  Group,
  Loader,
  Stack,
  Tabs,
  Text,
  Title,
} from "@mantine/core";
import { IconChevronLeft } from "@tabler/icons-react";
import { notifications } from "@mantine/notifications";
import { UserRole } from "@/lib/api";
import { ErrorState } from "@/components/ErrorState";
import { useUserAccessDetail } from "@/features/subjects/hooks/useUserAccessDetail";
import { useSetUserAccess } from "@/features/subjects/hooks/useSetUserAccess";
import { SubjectGroupsPanel } from "@/features/subjects/components/SubjectGroupsPanel";
import { EffectiveHostsPanel } from "@/features/subjects/components/EffectiveHostsPanel";
import { UserDevicesTab } from "@/features/subjects/components/UserDevicesTab";
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
import { DeleteUserModal } from "@/features/auth/components/DeleteUserModal";
import { RoleChangeModal } from "@/features/auth/components/RoleChangeModal";
import type { PendingRole } from "@/features/auth/components/RoleChangeModal";
import type { DeleteTarget } from "@/features/auth/components/DeleteUserModal";
import { toErrorMessage } from "@/lib/api-client";

function roleBadgeColor(role: UserRole): string {
  if (role === UserRole.SUPERADMIN) return "violet";
  if (role === UserRole.ADMIN) return "indigo";
  return "gray";
}

const UserDetailTab = {
  ACCESS: "access",
  DEVICES: "devices",
} as const;

type UserDetailTabValue = (typeof UserDetailTab)[keyof typeof UserDetailTab];
const VALID_USER_DETAIL_TABS = new Set<string>(Object.values(UserDetailTab));
function resolveTab(raw: string | null): UserDetailTabValue {
  return raw !== null && VALID_USER_DETAIL_TABS.has(raw)
    ? (raw as UserDetailTabValue)
    : UserDetailTab.ACCESS;
}

export function UserDetailPage() {
  const { id } = useParams();
  const navigate = useNavigate();
  const [searchParams, setSearchParams] = useSearchParams();
  const userId = Number(id);

  const { data, isPending, isError, error, refetch } = useUserAccessDetail(userId);
  const saveMutation = useSetUserAccess();

  const [draft, dispatch] = useReducer(
    subjectAccessReducer,
    undefined,
    initialSubjectAccessDraft,
  );

  const [deleteTarget, setDeleteTarget] = useState<DeleteTarget | null>(null);
  const [pendingRole, setPendingRole] = useState<PendingRole | null>(null);

  // Key the reset on the actual grant state to avoid resetting on background refetches
  const grantSignature = useMemo(() => {
    if (!data) return null;
    return (
      String(data.bypass_host_check) +
      "|" +
      data.groups
        .filter((g) => g.granted)
        .map((g) => g.id)
        .sort()
        .join(",")
    );
  }, [data]);

  useEffect(() => {
    if (data) dispatch({ type: "reset", groups: data.groups, bypassHostCheck: data.bypass_host_check });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [grantSignature]);

  const savedDraft = useMemo(
    () => (data ? initDraftFromGroups(data.groups, data.bypass_host_check) : null),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [grantSignature],
  );
  const dirty = savedDraft != null && isSubjectAccessDirty(draft, savedDraft);

  // Bypass grants this user's devices access to every host (including future
  // ones), warn about this and ensure user confirms
  const bypassJustEnabled = savedDraft != null && isBypassJustEnabled(savedDraft, draft);
  const bypassAckRequired = savedDraft != null && requiresBypassAcknowledgement(savedDraft, draft);

  const liveAddressCount = useMemo(
    () => data?.devices.reduce((sum, d) => sum + d.live_address_count, 0) ?? 0,
    [data],
  );

  useUnsavedChangesGuard(dirty);

  function handleSaveAccess() {
    if (bypassAckRequired) return;
    saveMutation.mutate(
      { path: { user_id: userId }, body: buildModifyAccessRequest(draft) },
      {
        onSuccess: () => notifications.show({ color: "green", message: "Access updated" }),
        onError: (err) => notifications.show({ color: "red", message: toErrorMessage(err) }),
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
        title="Could not load user"
        message={error ? undefined : "This user could not be loaded."}
        onRetry={() => refetch()}
      />
    );
  }

  const isSuperadmin = data.role === UserRole.SUPERADMIN;

  return (
    <>
      <DeleteUserModal
        deleteTarget={deleteTarget}
        onClose={() => {
          setDeleteTarget(null);
          if (deleteTarget) navigate(ROUTES.accessUsers);
        }}
      />
      <RoleChangeModal pendingRole={pendingRole} onClose={() => setPendingRole(null)} />

      <Stack maw={1200} gap="lg" pb={dirty ? STAGED_BAR_HEIGHT : undefined}>
        {/* Header */}
        <div>
          <Anchor
            component={Link}
            to={ROUTES.accessUsers}
            size="sm"
            c="dimmed"
            style={{ display: "inline-flex", alignItems: "center", minHeight: 24 }}
          >
            <Group gap={4}>
              <IconChevronLeft size={14} />
              Users
            </Group>
          </Anchor>

          <Group justify="space-between" align="flex-start" mt="xs">
            <Stack gap={2}>
              <Group gap="xs" align="baseline">
                <Title order={1}>{data.display_name}</Title>
                <Badge variant="outline" color={roleBadgeColor(data.role)} size="sm">
                  {data.role.toUpperCase()}
                </Badge>
              </Group>
              <Text size="sm" c="dimmed">
                {data.username}
              </Text>
            </Stack>

            {!isSuperadmin && (
              <Group gap="sm">
                <Button
                  size="xs"
                  variant="light"
                  color="indigo"
                  onClick={() =>
                    setPendingRole({
                      userId: data.id,
                      username: data.username,
                      targetRole: data.role === UserRole.ADMIN ? "user" : "admin",
                    })
                  }
                >
                  {data.role === UserRole.ADMIN ? "Demote to user" : "Promote to admin"}
                </Button>
                <Button
                  size="xs"
                  variant="light"
                  color="red"
                  onClick={() => setDeleteTarget({ id: data.id, username: data.username })}
                >
                  Delete
                </Button>
              </Group>
            )}
          </Group>
        </div>

        {/* Tabs */}
        <Tabs
          value={resolveTab(searchParams.get("tab"))}
          onChange={(value) =>
            setSearchParams(
              (prev) => {
                prev.set("tab", resolveTab(value));
                return prev;
              },
              { replace: true },
            )
          }
        >
          <Tabs.List>
            <Tabs.Tab value={UserDetailTab.ACCESS}>Access</Tabs.Tab>
            <Tabs.Tab value={UserDetailTab.DEVICES}>
              Devices{" "}
              <Text span size="xs" c="dimmed">
                {data.devices.length}
              </Text>
            </Tabs.Tab>
          </Tabs.List>

          <Tabs.Panel value={UserDetailTab.ACCESS} pt="md">
            <Grid>
              <Grid.Col span={{ base: 12, md: 5 }}>
                <SubjectGroupsPanel
                  groups={data.groups}
                  draft={draft}
                  dispatch={dispatch}
                  disabled={saveMutation.isPending}
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
          </Tabs.Panel>

          <Tabs.Panel value={UserDetailTab.DEVICES} pt="md">
            <UserDevicesTab userId={data.id} devices={data.devices} />
          </Tabs.Panel>
        </Tabs>

        <StagedChangesBar
          visible={dirty}
          summary={
            bypassJustEnabled
              ? "You're about to enable host-check bypass."
              : "You have unsaved access changes."
          }
          saving={saveMutation.isPending}
          onSave={handleSaveAccess}
          onDiscard={handleDiscardAccess}
          warning={
            bypassJustEnabled
              ? {
                  detail: `Enabling bypass lets ${data.display_name} reach all hosts, including future ones, from ${liveAddressCount} live ${liveAddressCount === 1 ? "address" : "addresses"} across their devices.`,
                  acknowledgeLabel: "I understand this exposes every host to this user.",
                  acknowledged: draft.bypassAcknowledged,
                  onAcknowledgeChange: (value) => dispatch({ type: "acknowledgeBypass", value }),
                }
              : undefined
          }
        />
      </Stack>
    </>
  );
}
