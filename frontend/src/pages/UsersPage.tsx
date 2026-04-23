import { useState } from "react";
import { useForm, schemaResolver } from "@mantine/form";
import {
  ActionIcon,
  Badge,
  Button,
  Card,
  Group,
  Modal,
  PasswordInput,
  Stack,
  Switch,
  Table,
  Text,
  TextInput,
  Title,
  Tooltip,
} from "@mantine/core";
import { notifications } from "@mantine/notifications";
import { IconEdit, IconTrash } from "@tabler/icons-react";
import { useAuth } from "@/features/auth/hooks/useAuth";
import { useListUsers } from "@/features/auth/hooks/useListUsers";
import { useCreateUser } from "@/features/auth/hooks/useCreateUser";
import { usePromoteUser } from "@/features/auth/hooks/usePromoteUser";
import { useDemoteUser } from "@/features/auth/hooks/useDemoteUser";
import { useDeleteUser } from "@/features/auth/hooks/useDeleteUser";
import { useUsersHostAccess } from "@/features/host-access/hooks/useUsersHostAccess";
import { UserAllowlistDrawer } from "@/features/host-access/components/UserAllowlistDrawer";
import { UserRole } from "@/lib/api";
import type { UserHostAccessSummary } from "@/lib/api";
import { toApiError, toErrorMessage } from "@/lib/api-client";
import { zCreateUserRequest, zPromoteUserRequest } from "@/lib/api/zod.gen";
import type { z } from "zod";

const createUserSchema = zCreateUserRequest;
type CreateUserValues = z.infer<typeof createUserSchema>;

function roleBadgeColor(role: UserRole): string {
  if (role === UserRole.SUPERADMIN) return "violet";
  if (role === UserRole.ADMIN) return "indigo";
  return "gray";
}

function AccessSummaryCell({ summary }: { summary: UserHostAccessSummary }) {
  if (summary.bypass) {
    return (
      <Group gap="xs">
        <Badge variant="light" color="gray" size="sm">
          Allow all
        </Badge>
        {summary.role === UserRole.USER && (
          <Badge color="red" size="xs">
            Risky
          </Badge>
        )}
      </Group>
    );
  }
  if (summary.direct_host_count === 0 && summary.groups.length === 0) {
    return (
      <Text size="sm" c="red" fw={500}>
        No access
      </Text>
    );
  }
  return (
    <Group gap="xs" wrap="wrap">
      {summary.direct_host_count > 0 && (
        <Badge variant="light" color="indigo" size="sm">
          {summary.direct_host_count} direct
        </Badge>
      )}
      {summary.groups.map((g) => (
        <Badge key={g.id} variant="light" color="indigo" size="sm">
          {g.name}
        </Badge>
      ))}
    </Group>
  );
}

export function UsersPage() {
  const { user: currentUser } = useAuth();
  const listUsers = useListUsers({ enabled: currentUser != null });
  const usersHostAccess = useUsersHostAccess();
  const promoteUser = usePromoteUser();
  const demoteUser = useDemoteUser();
  const deleteUser = useDeleteUser();
  const createUser = useCreateUser();

  const [drawerUser, setDrawerUser] = useState<UserHostAccessSummary | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<{ id: number; username: string } | null>(null);
  const [createModalOpen, setCreateModalOpen] = useState(false);
  const [pendingRole, setPendingRole] = useState<{
    userId: number;
    username: string;
    targetRole: "admin" | "user";
  } | null>(null);
  const [promotePassword, setPromotePassword] = useState("");
  const [promotePasswordError, setPromotePasswordError] = useState("");

  const createForm = useForm<CreateUserValues>({
    validate: schemaResolver(createUserSchema),
    initialValues: { username: "", email: "", display_name: "" },
  });

  // Build a lookup from user ID to host access summary
  const accessByUserId = new Map(
    (usersHostAccess.data ?? []).map((s) => [s.id, s]),
  );

  const users = listUsers.data ?? [];

  function handleCreateUser(values: CreateUserValues) {
    createUser.mutate(
      { body: values },
      {
        onSuccess: () => {
          notifications.show({ color: "green", message: "User created" });
          setCreateModalOpen(false);
          createForm.reset();
        },
        onError: (err) => {
          const message =
            toApiError(err).status === 409
              ? "A user with this username already exists."
              : toErrorMessage(err);
          notifications.show({ color: "red", title: "Failed to create user", message });
        },
      },
    );
  }

  function handleCloseCreateModal() {
    setCreateModalOpen(false);
    createForm.reset();
  }

  function handleRoleToggle(targetUserId: number, currentRole: string, username: string) {
    const targetRole = currentRole === UserRole.ADMIN ? "user" : "admin";
    setPromotePassword("");
    setPromotePasswordError("");
    setPendingRole({ userId: targetUserId, username, targetRole });
  }

  function handleConfirmRoleChange() {
    if (!pendingRole) return;
    if (pendingRole.targetRole === "admin") {
      const result = zPromoteUserRequest.safeParse({ password: promotePassword });
      if (!result.success) {
        setPromotePasswordError("Password must be at least 8 characters.");
        return;
      }
      promoteUser.mutate(
        { path: { user_id: pendingRole.userId }, body: { password: promotePassword } },
        {
          onSuccess: () =>
            notifications.show({ color: "green", message: "User promoted to admin" }),
          onError: (err) =>
            notifications.show({ color: "red", title: "Failed to promote user", message: toErrorMessage(err) }),
          onSettled: () => {
            setPendingRole(null);
            setPromotePassword("");
          },
        },
      );
    } else {
      demoteUser.mutate(
        { path: { user_id: pendingRole.userId } },
        {
          onSuccess: () =>
            notifications.show({ color: "green", message: "User demoted" }),
          onError: (err) =>
            notifications.show({ color: "red", title: "Failed to demote user", message: toErrorMessage(err) }),
          onSettled: () => setPendingRole(null),
        },
      );
    }
  }

  function confirmDeleteUser() {
    if (!deleteTarget) return;
    deleteUser.mutate(
      { path: { user_id: deleteTarget.id } },
      {
        onSuccess: () =>
          notifications.show({ color: "green", message: "User deleted" }),
        onError: (err) =>
          notifications.show({ color: "red", title: "Failed to delete user", message: toErrorMessage(err) }),
        onSettled: () => setDeleteTarget(null),
      },
    );
  }

  return (
    <>
      {/* Role change modal */}
      <Modal
        opened={pendingRole !== null}
        onClose={() => setPendingRole(null)}
        title={pendingRole?.targetRole === "admin" ? "Promote to admin?" : "Demote to user?"}
        closeOnClickOutside={false}
      >
        {pendingRole?.targetRole === "admin" ? (
          <Stack gap="sm">
            <Text size="sm">
              Promoting{" "}
              <Text component="span" fw={600}>
                {pendingRole?.username}
              </Text>{" "}
              to admin will give them login access and full visibility of all devices. Set an
              initial password they will use to log in.
            </Text>
            <PasswordInput
              label="Initial password"
              placeholder="Min. 8 characters"
              value={promotePassword}
              onChange={(e) => {
                setPromotePassword(e.currentTarget.value);
                setPromotePasswordError("");
              }}
              error={promotePasswordError}
            />
          </Stack>
        ) : (
          <Text size="sm">
            Demoting{" "}
            <Text component="span" fw={600}>
              {pendingRole?.username}
            </Text>{" "}
            to user will revoke their login access and invalidate all their active sessions.
          </Text>
        )}
        <Group justify="flex-end" mt="md" gap="sm">
          <Button variant="outline" onClick={() => setPendingRole(null)}>
            Cancel
          </Button>
          <Button
            onClick={handleConfirmRoleChange}
            disabled={promoteUser.isPending || demoteUser.isPending}
            loading={promoteUser.isPending || demoteUser.isPending}
          >
            Confirm
          </Button>
        </Group>
      </Modal>

      {/* Delete user modal */}
      <Modal
        opened={deleteTarget !== null}
        onClose={() => setDeleteTarget(null)}
        title="Delete user"
        closeOnClickOutside={false}
        closeOnEscape={false}
        withCloseButton={false}
      >
        <Text size="sm">
          Are you sure you want to delete{" "}
          <Text component="span" fw={600}>
            {deleteTarget?.username}
          </Text>
          ? This action cannot be undone.
        </Text>
        <Group justify="flex-end" mt="md" gap="sm">
          <Button variant="outline" onClick={() => setDeleteTarget(null)}>
            Cancel
          </Button>
          <Button color="red" onClick={confirmDeleteUser} disabled={deleteUser.isPending}>
            {deleteUser.isPending ? "Deleting..." : "Delete"}
          </Button>
        </Group>
      </Modal>

      {/* Create user modal */}
      <Modal opened={createModalOpen} onClose={handleCloseCreateModal} title="Create user">
        <form onSubmit={createForm.onSubmit(handleCreateUser)}>
          <Stack gap="sm">
            <TextInput
              label="Username"
              placeholder="e.g. jgarcia"
              description="Lowercase letters, numbers, hyphens, and underscores only"
              {...createForm.getInputProps("username")}
            />
            <TextInput
              label="Email"
              type="email"
              placeholder="e.g. juan@example.com"
              {...createForm.getInputProps("email")}
            />
            <TextInput
              label="Display name"
              placeholder="e.g. Juan Garcia"
              {...createForm.getInputProps("display_name")}
            />
            <Group justify="flex-end" mt="xs" gap="sm">
              <Button type="button" variant="outline" onClick={handleCloseCreateModal}>
                Cancel
              </Button>
              <Button type="submit" disabled={createUser.isPending}>
                {createUser.isPending ? "Creating..." : "Create user"}
              </Button>
            </Group>
          </Stack>
        </form>
      </Modal>

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
                  <Table.Th>Display name</Table.Th>
                  <Table.Th>Role</Table.Th>
                  <Table.Th>Host access</Table.Th>
                  <Table.Th>Allow all</Table.Th>
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
                        {u.username}
                        {isSelf && (
                          <Text component="span" c="dimmed" size="xs" ml="xs">
                            (you)
                          </Text>
                        )}
                      </Table.Td>
                      <Table.Td c="dimmed">{u.display_name || "—"}</Table.Td>
                      <Table.Td>
                        <Badge variant="light" color={roleBadgeColor(u.role)}>
                          {u.role}
                        </Badge>
                      </Table.Td>
                      <Table.Td>
                        {accessSummary ? (
                          <AccessSummaryCell summary={accessSummary} />
                        ) : (
                          <Text size="sm" c="dimmed">
                            —
                          </Text>
                        )}
                      </Table.Td>
                      <Table.Td>
                        {accessSummary && (
                          <Switch
                            checked={accessSummary.bypass}
                            readOnly
                            size="sm"
                            aria-label="Allow all hosts toggle — click Edit access to change"
                          />
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
                            <>
                              <Button
                                type="button"
                                variant="outline"
                                size="xs"
                                disabled={promoteUser.isPending || demoteUser.isPending}
                                onClick={() => handleRoleToggle(u.id, u.role, u.username)}
                              >
                                {isUserRole ? "Promote" : "Demote"}
                              </Button>
                              <Tooltip label="Delete user" withArrow>
                                <ActionIcon
                                  variant="subtle"
                                  color="red"
                                  disabled={deleteUser.isPending}
                                  onClick={() => setDeleteTarget({ id: u.id, username: u.username })}
                                  aria-label={`Delete user ${u.username}`}
                                >
                                  <IconTrash size={16} stroke={1.5} />
                                </ActionIcon>
                              </Tooltip>
                            </>
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
