import { useState } from "react";
import {
  ActionIcon,
  Badge,
  Button,
  Card,
  Group,
  Modal,
  Table,
  Text,
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
import { toErrorMessage } from "@/lib/api-client";
import { UserRole } from "@/lib/api";

export function UsersTab() {
  const { user } = useAuth();
  const listUsers = useListUsers({ enabled: user?.role === UserRole.ADMIN });
  const promoteUser = usePromoteUser();
  const demoteUser = useDemoteUser();
  const deleteUser = useDeleteUser();
  const [deleteTarget, setDeleteTarget] = useState<{ id: number; username: string } | null>(null);

  const users = listUsers.data ?? [];

  function handleRoleToggle(targetUserId: number, currentRole: string) {
    const mutation = currentRole === UserRole.ADMIN ? demoteUser : promoteUser;
    mutation.mutate(
      { path: { user_id: targetUserId } },
      {
        onSuccess: () => notifications.show({ color: "green", message: "User updated" }),
        onError: (err) =>
          notifications.show({ color: "red", title: "Failed to update user", message: toErrorMessage(err) }),
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

      <Card withBorder>
        <Title order={3} mb="xs">Users</Title>
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
                            onClick={() => handleRoleToggle(adminUser.id, adminUser.role)}
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
