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
  Table,
  Text,
  TextInput,
  Title,
  Tooltip,
} from "@mantine/core";
import { notifications } from "@mantine/notifications";
import { IconTrash } from "@tabler/icons-react";
import { useAuth } from "@/features/auth/hooks/useAuth";
import { useListUsers } from "@/features/auth/hooks/useListUsers";
import { usePromoteUser } from "@/features/auth/hooks/usePromoteUser";
import { useDemoteUser } from "@/features/auth/hooks/useDemoteUser";
import { useDeleteUser } from "@/features/auth/hooks/useDeleteUser";
import { useCreateUser } from "@/features/auth/hooks/useCreateUser";
import { toApiError, toErrorMessage } from "@/lib/api-client";
import { UserRole } from "@/lib/api";
import { zCreateUserRequest } from "@/lib/api/zod.gen";
import type { z } from "zod";

const createUserSchema = zCreateUserRequest;
type CreateUserValues = z.infer<typeof createUserSchema>;

export function UsersTab() {
  const { user } = useAuth();
  const listUsers = useListUsers({ enabled: user?.role === UserRole.ADMIN });
  const promoteUser = usePromoteUser();
  const demoteUser = useDemoteUser();
  const deleteUser = useDeleteUser();
  const createUser = useCreateUser();
  const [deleteTarget, setDeleteTarget] = useState<{ id: number; username: string } | null>(null);
  const [createModalOpen, setCreateModalOpen] = useState(false);
  const [pendingRole, setPendingRole] = useState<{
    userId: number;
    username: string;
    targetRole: "admin" | "user";
  } | null>(null);

  const createForm = useForm<CreateUserValues>({
    validate: schemaResolver(createUserSchema),
    initialValues: { username: "", email: "", display_name: "", password: "" },
  });

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

  const users = listUsers.data ?? [];

  function handleRoleToggle(targetUserId: number, currentRole: string, username: string) {
    const targetRole = currentRole === UserRole.ADMIN ? "user" : "admin";
    setPendingRole({ userId: targetUserId, username, targetRole });
  }

  function handleConfirmRoleChange() {
    if (!pendingRole) return;
    const mutation = pendingRole.targetRole === "admin" ? promoteUser : demoteUser;
    mutation.mutate(
      { path: { user_id: pendingRole.userId } },
      {
        onSuccess: () => notifications.show({ color: "green", message: "User updated" }),
        onError: (err) =>
          notifications.show({ color: "red", title: "Failed to update user", message: toErrorMessage(err) }),
        onSettled: () => setPendingRole(null),
      },
    );
  }

  function confirmDeleteUser() {
    if (!deleteTarget) return;
    deleteUser.mutate(
      { path: { user_id: deleteTarget.id } },
      {
        onSuccess: () => notifications.show({ color: "green", message: "User deleted" }),
        onError: (err) =>
          notifications.show({ color: "red", title: "Failed to delete user", message: toErrorMessage(err) }),
        onSettled: () => setDeleteTarget(null),
      },
    );
  }

  return (
    <>
      <Modal
        opened={pendingRole !== null}
        onClose={() => setPendingRole(null)}
        title={pendingRole?.targetRole === "admin" ? "Promote to admin?" : "Demote to user?"}
        closeOnClickOutside={false}
      >
        {pendingRole?.targetRole === "admin" ? (
          <Text size="sm">
            Promoting{" "}
            <Text component="span" fw={600}>{pendingRole.username}</Text> to admin
            will give them visibility of <Text component="span" fw={500}>all devices</Text> across all
            users. They will also gain access to admin-only pages: Dashboard,
            Access Log, and Address History, including server-wide metrics.
          </Text>
        ) : (
          <Text size="sm">
            Demoting{" "}
            <Text component="span" fw={600}>{pendingRole?.username}</Text> to user
            will restrict them to seeing <Text component="span" fw={500}>only their own devices</Text>.
            They will lose access to admin-only pages (Dashboard, Access Log,
            Address History) and will no longer be able to view server-wide
            metrics.
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
          <Text component="span" fw={600}>{deleteTarget?.username}</Text>?
          This action cannot be undone.
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

      <Modal
        opened={createModalOpen}
        onClose={handleCloseCreateModal}
        title="Create user"
      >
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
            <PasswordInput
              label="Temporary password"
              description="The user will be asked to change this on first login."
              {...createForm.getInputProps("password")}
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

      <Card withBorder>
        <Group justify="space-between" mb="xs">
          <Title order={3}>Users</Title>
          <Button size="sm" onClick={() => setCreateModalOpen(true)}>
            Create user
          </Button>
        </Group>
        <Text c="dimmed" size="sm" mb="md">
          Promote or demote users between the user and admin roles. Users manage their own profile information.
        </Text>
        <Table.ScrollContainer minWidth={500}>
          <Table>
            <Table.Thead>
              <Table.Tr>
                <Table.Th>Username</Table.Th>
                <Table.Th>Display name</Table.Th>
                <Table.Th>Role</Table.Th>
                <Table.Th style={{ textAlign: "right" }}>Actions</Table.Th>
              </Table.Tr>
            </Table.Thead>
            <Table.Tbody>
              {users.map((adminUser) => {
                const isSelf = adminUser.id === user?.id;
                const isAdmin = adminUser.role === UserRole.ADMIN;
                return (
                  <Table.Tr key={adminUser.id}>
                    <Table.Td fw={500}>
                      {adminUser.username}
                      {isSelf && (
                        <Text component="span" c="dimmed" size="xs" ml="xs">(you)</Text>
                      )}
                    </Table.Td>
                    <Table.Td c="dimmed">{adminUser.display_name || "\u2014"}</Table.Td>
                    <Table.Td>
                      <Badge variant="light" color={isAdmin ? "indigo" : "gray"}>
                        {adminUser.role}
                      </Badge>
                    </Table.Td>
                    <Table.Td>
                      {!isSelf && (
                        <Group justify="flex-end" gap="sm">
                          <Button
                            type="button"
                            variant="outline"
                            size="sm"
                            disabled={promoteUser.isPending || demoteUser.isPending}
                            onClick={() => handleRoleToggle(adminUser.id, adminUser.role, adminUser.username)}
                          >
                            {isAdmin ? "Demote to user" : "Promote to admin"}
                          </Button>
                          <Tooltip label="Delete user" withArrow>
                            <ActionIcon
                              variant="subtle"
                              color="red"
                              disabled={deleteUser.isPending}
                              onClick={() => setDeleteTarget({ id: adminUser.id, username: adminUser.username })}
                              aria-label={`Delete user ${adminUser.username}`}
                            >
                              <IconTrash size={16} stroke={1.5} />
                            </ActionIcon>
                          </Tooltip>
                        </Group>
                      )}
                    </Table.Td>
                  </Table.Tr>
                );
              })}
            </Table.Tbody>
          </Table>
        </Table.ScrollContainer>
      </Card>
    </>
  );
}
