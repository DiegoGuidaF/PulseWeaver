import { useState } from "react";
import {
  ActionIcon,
  Badge,
  Button,
  Card,
  Group,
  Menu,
  Stack,
  Table,
  Text,
  Title,
  Tooltip,
} from "@mantine/core";
import {
  IconArrowDown,
  IconArrowUp,
  IconDotsVertical,
  IconEdit,
  IconTrash,
} from "@tabler/icons-react";
import { useAuth } from "@/features/auth/hooks/useAuth";
import { useListUsers } from "@/features/auth/hooks/useListUsers";
import { useUsersHostAccess } from "@/features/host-access/hooks/useUsersHostAccess";
import { UserAllowlistDrawer } from "@/features/host-access/components/UserAllowlistDrawer";
import { GroupBadgeList } from "@/features/host-access/components/GroupBadgeList";
import { CreateUserModal } from "@/features/auth/components/CreateUserModal";
import { RoleChangeModal } from "@/features/auth/components/RoleChangeModal";
import { DeleteUserModal } from "@/features/auth/components/DeleteUserModal";
import type { PendingRole } from "@/features/auth/components/RoleChangeModal";
import type { DeleteTarget } from "@/features/auth/components/DeleteUserModal";
import { UserRole } from "@/lib/api";
import type { UserHostAccessSummary } from "@/lib/api";

function roleBadgeColor(role: UserRole): string {
  if (role === UserRole.SUPERADMIN) return "violet";
  if (role === UserRole.ADMIN) return "indigo";
  return "gray";
}

function IndividualHostsCell({ summary }: { summary: UserHostAccessSummary }) {
  if (summary.bypass) {
    return (
      <Badge variant="light" color="gray" size="sm">
        All hosts allowed
      </Badge>
    );
  }
  if (summary.direct_host_count === 0) {
    return (
      <Text size="sm" c="dimmed">
        —
      </Text>
    );
  }
  return (
    <Badge variant="light" color="indigo" size="sm">
      {summary.direct_host_count} {summary.direct_host_count === 1 ? "host" : "hosts"}
    </Badge>
  );
}

export function UsersPage() {
  const { user: currentUser } = useAuth();
  const listUsers = useListUsers({ enabled: currentUser != null });
  const usersHostAccess = useUsersHostAccess();

  const [drawerUser, setDrawerUser] = useState<UserHostAccessSummary | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<DeleteTarget | null>(null);
  const [createModalOpen, setCreateModalOpen] = useState(false);
  const [pendingRole, setPendingRole] = useState<PendingRole | null>(null);

  const accessByUserId = new Map(
    (usersHostAccess.data ?? []).map((s) => [s.id, s]),
  );

  const users = listUsers.data ?? [];

  function handleRoleToggle(userId: number, currentRole: string, username: string) {
    const targetRole = currentRole === UserRole.ADMIN ? "user" : "admin";
    setPendingRole({ userId, username, targetRole });
  }

  return (
    <>
      <CreateUserModal opened={createModalOpen} onClose={() => setCreateModalOpen(false)} />
      <RoleChangeModal pendingRole={pendingRole} onClose={() => setPendingRole(null)} />
      <DeleteUserModal deleteTarget={deleteTarget} onClose={() => setDeleteTarget(null)} />
      <UserAllowlistDrawer user={drawerUser} onClose={() => setDrawerUser(null)} />

      <Stack maw={1100} gap="xl">
        <Group justify="space-between" align="flex-end">
          <div>
            <Title order={1}>Users</Title>
            <Text c="dimmed" mt={4}>
              Manage users and control which hosts each user can reach.
            </Text>
          </div>
          <Button onClick={() => setCreateModalOpen(true)}>Create user</Button>
        </Group>

        <Card withBorder>
          <Table.ScrollContainer minWidth={700}>
            <Table>
              <Table.Thead>
                <Table.Tr>
                  <Table.Th>Username</Table.Th>
                  <Table.Th>Role</Table.Th>
                  <Table.Th>Individual Hosts</Table.Th>
                  <Table.Th>Groups</Table.Th>
                  <Table.Th style={{ textAlign: "right" }}>Actions</Table.Th>
                </Table.Tr>
              </Table.Thead>
              <Table.Tbody>
                {users.map((u) => {
                  const isSelf = u.id === currentUser?.id;
                  const isSuperadmin = u.role === UserRole.SUPERADMIN;
                  const isUserRole = u.role === UserRole.USER;
                  const accessSummary = accessByUserId.get(u.id);

                  return (
                    <Table.Tr key={u.id}>
                      <Table.Td fw={500}>
                        <Tooltip label={u.display_name || "—"} withArrow>
                          <span>
                            {u.username}
                            {isSelf && (
                              <Text component="span" c="dimmed" size="xs" ml="xs">
                                (you)
                              </Text>
                            )}
                          </span>
                        </Tooltip>
                      </Table.Td>
                      <Table.Td>
                        <Badge variant="light" color={roleBadgeColor(u.role)}>
                          {u.role}
                        </Badge>
                      </Table.Td>
                      <Table.Td>
                        {accessSummary ? (
                          <IndividualHostsCell summary={accessSummary} />
                        ) : (
                          <Text size="sm" c="dimmed">
                            —
                          </Text>
                        )}
                      </Table.Td>
                      <Table.Td>
                        {accessSummary && accessSummary.groups.length > 0 ? (
                          <GroupBadgeList groups={accessSummary.groups} size="sm" />
                        ) : (
                          <Text size="sm" c="dimmed">
                            —
                          </Text>
                        )}
                      </Table.Td>
                      <Table.Td>
                        <Group justify="flex-end" gap="xs">
                          {accessSummary && (
                            <Tooltip label="Edit host access" withArrow>
                              <ActionIcon
                                variant="subtle"
                                onClick={() => setDrawerUser(accessSummary)}
                                aria-label={`Edit host access for ${u.username}`}
                              >
                                <IconEdit size={16} stroke={1.5} />
                              </ActionIcon>
                            </Tooltip>
                          )}
                          {!isSelf && !isSuperadmin && (
                            <Menu position="bottom-end" withArrow shadow="md">
                              <Menu.Target>
                                <ActionIcon
                                  variant="subtle"
                                  aria-label={`More actions for ${u.username}`}
                                >
                                  <IconDotsVertical size={16} stroke={1.5} />
                                </ActionIcon>
                              </Menu.Target>
                              <Menu.Dropdown>
                                <Menu.Item
                                  leftSection={
                                    isUserRole ? (
                                      <IconArrowUp size={14} stroke={1.5} />
                                    ) : (
                                      <IconArrowDown size={14} stroke={1.5} />
                                    )
                                  }
                                  onClick={() => handleRoleToggle(u.id, u.role, u.username)}
                                >
                                  {isUserRole ? "Promote to admin" : "Demote to user"}
                                </Menu.Item>
                                <Menu.Divider />
                                <Menu.Item
                                  color="red"
                                  leftSection={<IconTrash size={14} stroke={1.5} />}
                                  onClick={() => setDeleteTarget({ id: u.id, username: u.username })}
                                >
                                  Delete user
                                </Menu.Item>
                              </Menu.Dropdown>
                            </Menu>
                          )}
                        </Group>
                      </Table.Td>
                    </Table.Tr>
                  );
                })}
              </Table.Tbody>
            </Table>
          </Table.ScrollContainer>
        </Card>
      </Stack>
    </>
  );
}
