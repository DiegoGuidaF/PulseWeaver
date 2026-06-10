import { useMemo, useState } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { buildRoute } from "@/lib/routes";
import {
  Badge,
  Button,
  Center,
  Group,
  Loader,
  Stack,
  Text,
  Title,
} from "@mantine/core";
import { DataTable, type DataTableSortStatus } from "mantine-datatable";
import type { GroupSummary, UserListItem } from "@/lib/api";
import { UserRole } from "@/lib/api";
import { useListUsersWithAccess } from "@/features/subjects/hooks/useListUsersWithAccess";
import { ErrorState } from "@/components/ErrorState";
import { GroupFilterBar } from "@/features/subjects/components/GroupFilterBar";
import { GroupBadgeList } from "@/features/host-access/components/GroupBadgeList";
import { formatEffectiveAccess } from "@/features/subjects/constants";
import { AllHostsBypassPill } from "@/features/subjects/components/AllHostsBypassPill";
import { CreateUserModal } from "@/features/auth/components/CreateUserModal";
function roleBadgeColor(role: UserRole): string {
  if (role === UserRole.SUPERADMIN) return "violet";
  if (role === UserRole.ADMIN) return "indigo";
  return "gray";
}

function collectGroups(users: UserListItem[]) {
  const seen = new Map<number, GroupSummary>();
  for (const u of users) {
    for (const g of u.groups) {
      if (!seen.has(g.id)) seen.set(g.id, g);
    }
  }
  return [...seen.values()].sort((a, b) => a.name.localeCompare(b.name));
}

export function UsersPage() {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const { data, isPending, isError, error, refetch } = useListUsersWithAccess();

  const [createModalOpen, setCreateModalOpen] = useState(false);
  const [groupFilter, setGroupFilter] = useState<Set<number>>(() => {
    const gid = searchParams.get("group_id");
    return gid ? new Set([Number(gid)]) : new Set();
  });
  const [sortStatus, setSortStatus] = useState<DataTableSortStatus<UserListItem>>({
    columnAccessor: "display_name",
    direction: "asc",
  });

  function toggleGroupFilter(groupId: number) {
    setGroupFilter((prev) => {
      const next = new Set(prev);
      if (next.has(groupId)) next.delete(groupId);
      else next.add(groupId);
      return next;
    });
  }

  const users = useMemo(() => data ?? [], [data]);
  const allGroups = useMemo(() => collectGroups(users), [users]);

  const displayedUsers = useMemo(() => {
    let list = users;

    if (groupFilter.size > 0) {
      list = list.filter((u) => u.groups.some((g) => groupFilter.has(g.id)));
    }

    const { columnAccessor, direction } = sortStatus;
    const mult = direction === "asc" ? 1 : -1;
    return [...list].sort((a, b) => {
      switch (columnAccessor) {
        case "display_name": return mult * a.display_name.localeCompare(b.display_name);
        case "host_count": return mult * (a.host_count - b.host_count);
        case "device_count": return mult * (a.device_count - b.device_count);
        case "live_address_count": return mult * (a.live_address_count - b.live_address_count);
        default: return 0;
      }
    });
  }, [users, groupFilter, sortStatus]);

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
          <ErrorState error={error} title="Failed to load users" onRetry={() => refetch()} />
        )}

        <GroupFilterBar
          availableGroups={allGroups}
          selected={groupFilter}
          onChange={setGroupFilter}
        />

        <DataTable
          records={displayedUsers}
          highlightOnHover
          onRowClick={({ record }) => navigate(buildRoute.accessUserDetail(record.id))}
          sortStatus={sortStatus}
          onSortStatusChange={setSortStatus}
          columns={[
            {
              accessor: "display_name",
              title: "Name",
              sortable: true,
              render: (u) => (
                <div>
                  <Text size="sm" fw={600}>{u.display_name}</Text>
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
                  <GroupBadgeList
                    groups={u.groups}
                    size="xs"
                    selected={groupFilter}
                    onGroupClick={toggleGroupFilter}
                  />
                ),
            },
            {
              accessor: "host_count",
              title: "Effective access",
              sortable: true,
              render: (u) => {
                if (u.bypass_host_check) {
                  return <AllHostsBypassPill />;
                }
                const text = formatEffectiveAccess(u);
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
              accessor: "live_address_count",
              title: "Live IPs",
              sortable: true,
              render: (u) => <Text size="sm" c="dimmed">{u.live_address_count}</Text>,
            },
          ]}
        />
      </Stack>
    </>
  );
}
