import { useMemo, useState } from "react";
import { useForm } from "@mantine/form";
import { zod4Resolver } from "mantine-form-zod-resolver";
import { z } from "zod";
import {
  Button,
  Card,
  Group,
  Modal,
  Stack,
  Table,
  Text,
  TextInput,
  Title,
} from "@mantine/core";
import { useAuth } from "@/features/auth/AuthContext";
import { useAdminUpdateUser } from "@/features/auth/hooks/useAdminUpdateUser";
import { useChangePassword } from "@/features/auth/hooks/useChangePassword";
import { useDeleteUser } from "@/features/auth/hooks/useDeleteUser";
import { useListUsers } from "@/features/auth/hooks/useListUsers";
import { useUpdateMe } from "@/features/auth/hooks/useUpdateMe";
import { UserRole } from "@/lib/api";

const profileSchema = z.object({
  display_name: z.string().trim().min(1).max(50).optional(),
  username: z.string().trim().min(3).max(32).regex(/^[a-zA-Z0-9_-]+$/).optional(),
  email: z.string().email().optional().or(z.literal("")),
});

const passwordSchema = z.object({
  current_password: z.string().min(1, "Current password is required"),
  password: z.string().min(8).max(72),
});

export function SettingsPage() {
  const { user } = useAuth();
  const updateMe = useUpdateMe();
  const changePassword = useChangePassword();
  const listUsers = useListUsers({ enabled: user?.role === UserRole.ADMIN });
  const adminUpdateUser = useAdminUpdateUser();
  const deleteUser = useDeleteUser();
  const [displayNameEdits, setDisplayNameEdits] = useState<Record<number, string>>({});
  const [deleteTarget, setDeleteTarget] = useState<{ id: number; username: string } | null>(null);

  const profileForm = useForm<z.infer<typeof profileSchema>>({
    validate: zod4Resolver(profileSchema),
    initialValues: {
      display_name: user?.display_name ?? "",
      username: user?.username ?? "",
      email: user?.email ?? "",
    },
  });

  const passwordForm = useForm<z.infer<typeof passwordSchema>>({
    validate: zod4Resolver(passwordSchema),
    initialValues: {
      current_password: "",
      password: "",
    },
  });

  const adminUsers = useMemo(() => listUsers.data ?? [], [listUsers.data]);

  function submitProfile(values: z.infer<typeof profileSchema>) {
    const body: { display_name?: string; username?: string; email?: string } = {};
    const nextDisplayName = values.display_name?.trim() ?? "";
    const nextUsername = values.username?.trim() ?? "";
    const nextEmail = values.email?.trim() ?? "";

    if (nextDisplayName && nextDisplayName !== user?.display_name) {
      body.display_name = nextDisplayName;
    }
    if (nextUsername && nextUsername !== user?.username) {
      body.username = nextUsername;
    }
    if (nextEmail && nextEmail !== user?.email) {
      body.email = nextEmail;
    }

    updateMe.mutate({ body });
  }

  function submitPassword(values: z.infer<typeof passwordSchema>) {
    changePassword.mutate(
      {
        body: {
          current_password: values.current_password,
          password: values.password,
        },
      },
      {
        onSuccess: () => passwordForm.reset(),
      },
    );
  }

  function handleRoleToggle(targetUserId: number, currentRole: string) {
    const nextRole = currentRole === UserRole.ADMIN ? UserRole.USER : UserRole.ADMIN;
    adminUpdateUser.mutate({
      path: { user_id: targetUserId },
      body: { role: nextRole },
    });
  }

  function handleAdminDisplayNameSave(targetUserId: number) {
    const displayName = displayNameEdits[targetUserId]?.trim();
    if (!displayName) return;
    adminUpdateUser.mutate({
      path: { user_id: targetUserId },
      body: { display_name: displayName },
    });
  }

  function handleDeleteUser(targetUserId: number, username: string) {
    setDeleteTarget({ id: targetUserId, username });
  }

  function confirmDeleteUser() {
    if (!deleteTarget) return;
    deleteUser.mutate(
      { path: { user_id: deleteTarget.id } },
      { onSettled: () => setDeleteTarget(null) },
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

      <Stack maw={1024} gap="xl">
        <div>
          <Title order={1}>Settings</Title>
          <Text c="dimmed">Manage your profile, password, and users.</Text>
        </div>

        {user?.must_change_password && (
          <Card withBorder style={{ borderColor: "var(--mantine-color-yellow-6)" }}>
            <Title order={4} mb="xs">Password change required</Title>
            <Text size="sm" c="dimmed">
              You must set a new password before using the rest of the application.
            </Text>
          </Card>
        )}

        <Card withBorder>
          <Title order={3} mb="md">My profile</Title>
          <form onSubmit={profileForm.onSubmit(submitProfile)}>
            <Stack gap="md">
              <TextInput
                label="Display name"
                placeholder="Your display name"
                {...profileForm.getInputProps("display_name")}
              />
              <TextInput
                label="Username"
                placeholder="Username"
                {...profileForm.getInputProps("username")}
              />
              <TextInput
                label="Email"
                placeholder="you@example.com"
                {...profileForm.getInputProps("email")}
              />
              <div>
                <Button type="submit" disabled={updateMe.isPending}>
                  {updateMe.isPending ? "Saving..." : "Save profile"}
                </Button>
              </div>
            </Stack>
          </form>
        </Card>

        <Card withBorder>
          <Title order={3} mb="md">Change password</Title>
          <form onSubmit={passwordForm.onSubmit(submitPassword)}>
            <Stack gap="md">
              <TextInput
                label="Current password"
                type="password"
                autoComplete="current-password"
                {...passwordForm.getInputProps("current_password")}
              />
              <TextInput
                label="New password"
                type="password"
                autoComplete="new-password"
                {...passwordForm.getInputProps("password")}
              />
              <div>
                <Button type="submit" disabled={changePassword.isPending}>
                  {changePassword.isPending ? "Updating..." : "Update password"}
                </Button>
              </div>
            </Stack>
          </form>
        </Card>

        {user?.role === UserRole.ADMIN && !user.must_change_password && (
          <Card withBorder>
            <Title order={3} mb="md">Users (admin)</Title>
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
                {adminUsers.map((adminUser) => {
                  const isSelf = adminUser.id === user.id;
                  return (
                    <Table.Tr key={adminUser.id}>
                      <Table.Td fw={500}>{adminUser.username}</Table.Td>
                      <Table.Td>
                        <Group gap="sm">
                          <TextInput
                            value={displayNameEdits[adminUser.id] ?? adminUser.display_name}
                            onChange={(event) =>
                              setDisplayNameEdits((prev) => ({
                                ...prev,
                                [adminUser.id]: event.target.value,
                              }))
                            }
                            disabled={isSelf || adminUpdateUser.isPending}
                            style={{ flex: 1 }}
                          />
                          <Button
                            type="button"
                            variant="outline"
                            size="sm"
                            disabled={isSelf || adminUpdateUser.isPending}
                            onClick={() => handleAdminDisplayNameSave(adminUser.id)}
                          >
                            Save
                          </Button>
                        </Group>
                      </Table.Td>
                      <Table.Td>{adminUser.role}</Table.Td>
                      <Table.Td>
                        <Group justify="flex-end" gap="sm">
                          <Button
                            type="button"
                            variant="outline"
                            size="sm"
                            disabled={isSelf || adminUpdateUser.isPending}
                            onClick={() => handleRoleToggle(adminUser.id, adminUser.role)}
                          >
                            {adminUser.role === UserRole.ADMIN ? "Demote" : "Promote"}
                          </Button>
                          <Button
                            type="button"
                            color="red"
                            size="sm"
                            disabled={isSelf || deleteUser.isPending}
                            onClick={() => handleDeleteUser(adminUser.id, adminUser.username)}
                          >
                            Delete
                          </Button>
                        </Group>
                      </Table.Td>
                    </Table.Tr>
                  );
                })}
              </Table.Tbody>
            </Table>
          </Card>
        )}
      </Stack>
    </>
  );
}
