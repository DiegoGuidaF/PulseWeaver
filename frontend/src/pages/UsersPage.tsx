import { useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import {
  ActionIcon,
  Badge,
  Button,
  Center,
  Group,
  Loader,
  Stack,
  Text,
  Title,
  Tooltip,
} from "@mantine/core";
import { IconArrowDown, IconArrowUp, IconTrash } from "@tabler/icons-react";
import { DataTable, type DataTableSortStatus } from "mantine-datatable";
import type { UserListItem } from "@/lib/api";
import { UserRole } from "@/lib/api";
import { useAuth } from "@/features/auth/hooks/useAuth";
import { useListUsersWithAccess } from "@/features/subjects/hooks/useListUsersWithAccess";
import { GroupFilterBar } from "@/features/subjects/components/GroupFilterBar";
import { formatEffectiveAccess } from "@/features/subjects/constants";
import { CreateUserModal } from "@/features/auth/components/CreateUserModal";
import { RoleChangeModal } from "@/features/auth/components/RoleChangeModal";
import { DeleteUserModal } from "@/features/auth/components/DeleteUserModal";
import type { PendingRole } from "@/features/auth/components/RoleChangeModal";
import type { DeleteTarget } from "@/features/auth/components/DeleteUserModal";
function roleBadgeColor(role: UserRole): string {
  if (role === UserRole.SUPERADMIN) return "violet";
  if (role === UserRole.ADMIN) return "indigo";
  return "gray";
}

function collectGroups(users: UserListItem[]) {
  const seen = new Map<number, { id: number; name: string }>();
  for (const u of users) {
    for (const g of u.groups) {
      if (!seen.has(g.id)) seen.set(g.id, g);
    }
  }
  return [...seen.values()].sort((a, b) => a.name.localeCompare(b.name));
}

export function UsersPage() {
  const navigate = useNavigate();
  const { user: currentUser } = useAuth();
  const { data, isPending, isError } = useListUsersWithAccess();

  const [createModalOpen, setCreateModalOpen] = useState(false);
  const [pendingRole, setPendingRole] = useState<PendingRole | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<DeleteTarget | null>(null);
  const [groupFilter, setGroupFilter] = useState<Set<number>>(new Set());
  const [sortStatus, setSortStatus] = useState<DataTableSortStatus<UserListItem>>({
    columnAccessor: "display_name",
    direction: "asc",
  });

  const users = useMemo(() => data ?? [], [data]);
  const allGroups = useMemo(() => collectGroups(users), [users]);

  const displayedUsers = useMemo(() => {
    let list = users;

    if (groupFilter.size > 0) {
      list = list.filter((u) => u.groups.some((g) => groupFilter.has(g.id)));
    }

    const superadmins = list.filter((u) => u.role === UserRole.SUPERADMIN);
    const rest = list.filter((u) => u.role !== UserRole.SUPERADMIN);

    const { columnAccessor, direction } = sortStatus;
    const mult = direction === "asc" ? 1 : -1;
    const sorted = [...rest].sort((a, b) => {
      switch (columnAccessor) {
        case "display_name": return mult * a.display_name.localeCompare(b.display_name);
        case "host_count": return mult * (a.host_count - b.host_count);
        case "device_count": return mult * (a.device_count - b.device_count);
        case "live_ip_count": return mult * (a.live_ip_count - b.live_ip_count);
        default: return 0;
      }
    });

    return [...superadmins, ...sorted];
  }, [users, groupFilter, sortStatus]);

  function handleRoleToggle(userId: number, currentRole: string, username: string) {
    const targetRole = currentRole === UserRole.ADMIN ? "user" : "admin";
    setPendingRole({ userId, username, targetRole });
  }

  if (isPending) {
    return (
      <Center py="xl">
        <Loader />
      </Center>
    );
  }

  return (
    <>
      <CreateUserModal opened={createModalOpen} onClose={() => setCreateModalOpen(false)} />
      <RoleChangeModal pendingRole={pendingRole} onClose={() => setPendingRole(null)} />
      <DeleteUserModal deleteTarget={deleteTarget} onClose={() => setDeleteTarget(null)} />

      <Stack maw={1200} gap="md">
        <Group justify="space-between" align="flex-start">
          <div>
            <Title order={1}>Users</Title>
            <Text c="dimmed" mt={4}>
              Manage access subjects and the hosts each can reach.
            </Text>
          </div>
          <Button onClick={() => setCreateModalOpen(true)}>+ New user</Button>
        </Group>

        {isError && (
          <Text c="red" size="sm">Failed to load users.</Text>
        )}

        <GroupFilterBar
          availableGroups={allGroups}
          selected={groupFilter}
          onChange={setGroupFilter}
        />

        <DataTable
          records={displayedUsers}
          highlightOnHover
          onRowClick={({ record }) => {
            if (record.role !== UserRole.SUPERADMIN) {
              navigate(`/access/users/${record.id}`);
            }
          }}
          sortStatus={sortStatus}
          onSortStatusChange={setSortStatus}
          rowStyle={(r) => (r.role === UserRole.SUPERADMIN ? { opacity: 0.7 } : undefined)}
          columns={[
            {
              accessor: "display_name",
              title: "Name",
              sortable: true,
              render: (u) => (
                <div>
                  <Group gap="xs" wrap="nowrap" align="baseline">
                    <Text size="sm" fw={600}>{u.display_name}</Text>
                    {u.role === UserRole.SUPERADMIN && (
                      <Badge size="xs" color="violet" variant="outline">superadmin</Badge>
                    )}
                  </Group>
                  <Text size="xs" c="dimmed">{u.username}</Text>
                </div>
              ),
            },
            {
              accessor: "role",
              title: "Role",
              render: (u) => (
                <Badge variant="light" color={roleBadgeColor(u.role)} size="sm">
                  {u.role}
                </Badge>
              ),
            },
            {
              accessor: "groups",
              title: "Groups",
              render: (u) =>
                u.groups.length === 0 ? (
                  <Text size="sm" c="dimmed">—</Text>
                ) : (
                  <Group gap={4} wrap="wrap">
                    {u.groups.map((g) => (
                      <Badge
                        key={g.id}
                        size="xs"
                        variant={groupFilter.has(g.id) ? "filled" : "outline"}
                        color="indigo"
                        style={{ cursor: "pointer" }}
                        onClick={(e) => {
                          e.stopPropagation();
                          const next = new Set(groupFilter);
                          if (next.has(g.id)) next.delete(g.id);
                          else next.add(g.id);
                          setGroupFilter(next);
                        }}
                      >
                        {g.name}
                      </Badge>
                    ))}
                  </Group>
                ),
            },
            {
              accessor: "host_count",
              title: "Effective access",
              sortable: true,
              render: (u) => {
                if (u.role === UserRole.SUPERADMIN) {
                  return <Text size="sm" c="dimmed">—</Text>;
                }
                const text = formatEffectiveAccess(u);
                if (u.bypass_host_check) {
                  return <Badge size="sm" color="orange" variant="light">{text}</Badge>;
                }
                return (
                  <Text size="sm" c={u.host_count === 0 ? "dimmed" : undefined}>
                    {text}
                  </Text>
                );
              },
            },
            {
              accessor: "device_count",
              title: "Devices",
              sortable: true,
              render: (u) => <Text size="sm" c="dimmed">{u.device_count}</Text>,
            },
            {
              accessor: "live_ip_count",
              title: "Live IPs",
              sortable: true,
              render: (u) => <Text size="sm" c="dimmed">{u.live_ip_count}</Text>,
            },
            {
              accessor: "actions",
              title: "",
              width: 80,
              render: (u) => {
                const isSelf = u.id === currentUser?.id;
                const isSuperadmin = u.role === UserRole.SUPERADMIN;
                if (isSelf || isSuperadmin) return null;
                const isUserRole = u.role === UserRole.USER;
                return (
                  <Group gap="xs" wrap="nowrap" justify="flex-end">
                    <Tooltip
                      label={isUserRole ? "Promote to admin" : "Demote to user"}
                      withArrow
                    >
                      <ActionIcon
                        variant="subtle"
                        size="sm"
                        onClick={(e) => {
                          e.stopPropagation();
                          handleRoleToggle(u.id, u.role, u.username);
                        }}
                        aria-label={isUserRole ? "Promote to admin" : "Demote to user"}
                      >
                        {isUserRole ? (
                          <IconArrowUp size={14} stroke={1.5} />
                        ) : (
                          <IconArrowDown size={14} stroke={1.5} />
                        )}
                      </ActionIcon>
                    </Tooltip>
                    <Tooltip label="Delete user" withArrow>
                      <ActionIcon
                        variant="subtle"
                        color="red"
                        size="sm"
                        onClick={(e) => {
                          e.stopPropagation();
                          setDeleteTarget({ id: u.id, username: u.username });
                        }}
                        aria-label={`Delete ${u.username}`}
                      >
                        <IconTrash size={14} stroke={1.5} />
                      </ActionIcon>
                    </Tooltip>
                  </Group>
                );
              },
            },
          ]}
        />
      </Stack>
    </>
  );
}
