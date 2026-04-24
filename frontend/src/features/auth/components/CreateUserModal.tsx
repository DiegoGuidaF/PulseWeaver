import { useForm, schemaResolver } from "@mantine/form";
import { Button, Group, Modal, Stack, TextInput } from "@mantine/core";
import { notifications } from "@mantine/notifications";
import { useCreateUser } from "@/features/auth/hooks/useCreateUser";
import { toApiError, toErrorMessage } from "@/lib/api-client";
import { zCreateUserRequest } from "@/lib/api/zod.gen";
import type { z } from "zod";

type CreateUserValues = z.infer<typeof zCreateUserRequest>;

interface Props {
  opened: boolean;
  onClose: () => void;
}

export function CreateUserModal({ opened, onClose }: Props) {
  const createUser = useCreateUser();
  const form = useForm<CreateUserValues>({
    validate: schemaResolver(zCreateUserRequest),
    initialValues: { username: "", email: "", display_name: "" },
  });

  function handleClose() {
    form.reset();
    onClose();
  }

  function handleSubmit(values: CreateUserValues) {
    createUser.mutate(
      { body: values },
      {
        onSuccess: () => {
          notifications.show({ color: "green", message: "User created" });
          handleClose();
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

  return (
    <Modal opened={opened} onClose={handleClose} title="Create user">
      <form onSubmit={form.onSubmit(handleSubmit)}>
        <Stack gap="sm">
          <TextInput
            label="Username"
            placeholder="e.g. jgarcia"
            description="Lowercase letters, numbers, hyphens, and underscores only"
            {...form.getInputProps("username")}
          />
          <TextInput
            label="Email"
            type="email"
            placeholder="e.g. juan@example.com"
            {...form.getInputProps("email")}
          />
          <TextInput
            label="Display name"
            placeholder="e.g. Juan Garcia"
            {...form.getInputProps("display_name")}
          />
          <Group justify="flex-end" mt="xs" gap="sm">
            <Button type="button" variant="outline" onClick={handleClose}>
              Cancel
            </Button>
            <Button type="submit" disabled={createUser.isPending} loading={createUser.isPending}>
              Create user
            </Button>
          </Group>
        </Stack>
      </form>
    </Modal>
  );
}
