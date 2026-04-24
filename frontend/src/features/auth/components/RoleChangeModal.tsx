import { useState } from "react";
import { Button, Group, Modal, PasswordInput, Stack, Text } from "@mantine/core";
import { notifications } from "@mantine/notifications";
import { usePromoteUser } from "@/features/auth/hooks/usePromoteUser";
import { useDemoteUser } from "@/features/auth/hooks/useDemoteUser";
import { toErrorMessage } from "@/lib/api-client";
import { zPromoteUserRequest } from "@/lib/api/zod.gen";

export interface PendingRole {
  userId: number;
  username: string;
  targetRole: "admin" | "user";
}

interface Props {
  pendingRole: PendingRole | null;
  onClose: () => void;
}

export function RoleChangeModal({ pendingRole, onClose }: Props) {
  const promoteUser = usePromoteUser();
  const demoteUser = useDemoteUser();
  const [password, setPassword] = useState("");
  const [passwordError, setPasswordError] = useState("");

  function handleClose() {
    setPassword("");
    setPasswordError("");
    onClose();
  }

  function handleConfirm() {
    if (!pendingRole) return;

    if (pendingRole.targetRole === "admin") {
      const result = zPromoteUserRequest.safeParse({ password });
      if (!result.success) {
        setPasswordError("Password must be at least 8 characters.");
        return;
      }
      promoteUser.mutate(
        { path: { user_id: pendingRole.userId }, body: { password } },
        {
          onSuccess: () =>
            notifications.show({ color: "green", message: "User promoted to admin" }),
          onError: (err) =>
            notifications.show({ color: "red", title: "Failed to promote user", message: toErrorMessage(err) }),
          onSettled: handleClose,
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
          onSettled: handleClose,
        },
      );
    }
  }

  return (
    <Modal
      opened={pendingRole !== null}
      onClose={handleClose}
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
            value={password}
            onChange={(e) => {
              setPassword(e.currentTarget.value);
              setPasswordError("");
            }}
            error={passwordError}
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
        <Button variant="outline" onClick={handleClose}>
          Cancel
        </Button>
        <Button
          onClick={handleConfirm}
          disabled={promoteUser.isPending || demoteUser.isPending}
          loading={promoteUser.isPending || demoteUser.isPending}
        >
          Confirm
        </Button>
      </Group>
    </Modal>
  );
}
