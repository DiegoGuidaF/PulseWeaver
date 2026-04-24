import { Button, Group, Modal, Text } from "@mantine/core";
import { notifications } from "@mantine/notifications";
import { useDeleteUser } from "@/features/auth/hooks/useDeleteUser";
import { toErrorMessage } from "@/lib/api-client";

export interface DeleteTarget {
  id: number;
  username: string;
}

interface Props {
  deleteTarget: DeleteTarget | null;
  onClose: () => void;
}

export function DeleteUserModal({ deleteTarget, onClose }: Props) {
  const deleteUser = useDeleteUser();

  function handleConfirm() {
    if (!deleteTarget) return;
    deleteUser.mutate(
      { path: { user_id: deleteTarget.id } },
      {
        onSuccess: () =>
          notifications.show({ color: "green", message: "User deleted" }),
        onError: (err) =>
          notifications.show({ color: "red", title: "Failed to delete user", message: toErrorMessage(err) }),
        onSettled: onClose,
      },
    );
  }

  return (
    <Modal
      opened={deleteTarget !== null}
      onClose={onClose}
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
        <Button variant="outline" onClick={onClose}>
          Cancel
        </Button>
        <Button color="red" onClick={handleConfirm} disabled={deleteUser.isPending} loading={deleteUser.isPending}>
          Delete
        </Button>
      </Group>
    </Modal>
  );
}
