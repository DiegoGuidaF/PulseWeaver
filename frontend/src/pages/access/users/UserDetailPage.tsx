import { useEffect, useMemo, useReducer, useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import {
  Alert,
  Anchor,
  Badge,
  Center,
  Grid,
  Group,
  Loader,
  Stack,
  Table,
  Tabs,
  Text,
  Title,
} from "@mantine/core";
import { IconAlertCircle, IconChevronLeft } from "@tabler/icons-react";
import { notifications } from "@mantine/notifications";
import { UserRole } from "@/lib/api";
import { useUserAccessDetail } from "@/features/subjects/hooks/useUserAccessDetail";
import { useSetUserAccess } from "@/features/subjects/hooks/useSetUserAccess";
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

export function UserDetailPage() {
  const { id } = useParams();
  const navigate = useNavigate();
  const userId = Number(id);

  const { data, isPending, isError } = useUserAccessDetail(userId);
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

  useUnsavedChangesGuard(dirty);

  function handleSaveAccess() {
    saveMutation.mutate(
      { path: { user_id: userId }, body: buildModifyAccessRequest(draft) },
      {
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
      <Alert icon={<IconAlertCircle size={16} />} color="red" title="Not found">
        This user could not be loaded.
      </Alert>
    );
  }

  const isSuperadmin = data.role === UserRole.SUPERADMIN;

  return (
    <>
      <DeleteUserModal
        deleteTarget={deleteTarget}
        onClose={() => {
          setDeleteTarget(null);
          if (deleteTarget) navigate("/access/users");
        }}
      />
      <RoleChangeModal pendingRole={pendingRole} onClose={() => setPendingRole(null)} />

      <Stack maw={1200} gap="lg" pb={dirty ? STAGED_BAR_HEIGHT : undefined}>
        {/* Header */}
        <div>
          <Anchor component={Link} to="/access/users" size="sm" c="dimmed">
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
                {data.username} · {data.devices.length}{" "}
                {data.devices.length === 1 ? "device" : "devices"}
              </Text>
            </Stack>

            {!isSuperadmin && (
              <Group gap="sm">
                <Badge
                  size="sm"
                  variant="outline"
                  color="indigo"
                  style={{ cursor: "pointer" }}
                  onClick={() =>
                    setPendingRole({
                      userId: data.id,
                      username: data.username,
                      targetRole: data.role === UserRole.ADMIN ? "user" : "admin",
                    })
                  }
                >
                  {data.role === UserRole.ADMIN ? "Demote to user" : "Promote to admin"}
                </Badge>
                <Badge
                  size="sm"
                  variant="outline"
                  color="red"
                  style={{ cursor: "pointer" }}
                  onClick={() =>
                    setDeleteTarget({ id: data.id, username: data.username })
                  }
                >
                  Delete
                </Badge>
              </Group>
            )}
          </Group>
        </div>

        {/* Tabs */}
        <Tabs defaultValue="access">
          <Tabs.List>
            <Tabs.Tab value="access">Access</Tabs.Tab>
            <Tabs.Tab value="devices">
              Devices{" "}
              <Text span size="xs" c="dimmed">
                {data.devices.length}
              </Text>
            </Tabs.Tab>
          </Tabs.List>

          <Tabs.Panel value="access" pt="md">
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

          <Tabs.Panel value="devices" pt="md">
            {data.devices.length === 0 ? (
              <Text size="sm" c="dimmed">No devices registered.</Text>
            ) : (
              <Stack gap="sm">
                <Table fz="sm" withRowBorders highlightOnHover>
                  <Table.Thead>
                    <Table.Tr>
                      <Table.Th>Device name</Table.Th>
                      <Table.Th>Live IPs</Table.Th>
                      <Table.Th>API key</Table.Th>
                    </Table.Tr>
                  </Table.Thead>
                  <Table.Tbody>
                    {data.devices.map((device) => (
                      <Table.Tr
                        key={device.id}
                        style={{ cursor: "pointer" }}
                        onClick={() => navigate(`/user-devices/${data.id}?device=${device.id}`)}
                      >
                        <Table.Td fw={500}>{device.name}</Table.Td>
                        <Table.Td c="dimmed">{device.live_ip_count}</Table.Td>
                        <Table.Td>
                          {device.api_key_prefix ? (
                            <Badge size="xs" variant="light" color="orange" ff="monospace">
                              ● {device.api_key_prefix}…
                            </Badge>
                          ) : (
                            <Text size="sm" c="dimmed">—</Text>
                          )}
                        </Table.Td>
                      </Table.Tr>
                    ))}
                  </Table.Tbody>
                </Table>
                <Anchor
                  component={Link}
                  to={`/user-devices/${data.id}`}
                  size="xs"
                  c="dimmed"
                >
                  All devices →
                </Anchor>
              </Stack>
            )}
          </Tabs.Panel>
        </Tabs>

        <StagedChangesBar
          visible={dirty}
          summary="You have unsaved access changes."
          saving={saveMutation.isPending}
          onSave={handleSaveAccess}
          onDiscard={handleDiscardAccess}
        />
      </Stack>
    </>
  );
}
