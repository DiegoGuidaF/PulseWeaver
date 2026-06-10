import { useForm } from "@mantine/form";
import { Alert, Button, Group, Modal, Stack, Text, TextInput } from "@mantine/core";
import { notifications } from "@mantine/notifications";
import { useNavigate } from "react-router-dom";
import { useCreateUser } from "@/features/auth/hooks/useCreateUser";
import { toApiError, toErrorMessage } from "@/lib/api-client";
import { buildRoute } from "@/lib/routes";

interface CreateUserValues {
  username: string;
  display_name: string;
  email: string;
}

interface Props {
  opened: boolean;
  onClose: () => void;
}

export function CreateUserModal({ opened, onClose }: Props) {
  const createUser = useCreateUser();
  const navigate = useNavigate();
  const form = useForm<CreateUserValues>({
    validateInputOnBlur: true,
    initialValues: { username: "", email: "", display_name: "" },
    validate: {
      username: (value) => {
        if (value.length < 3) return "Username must be at least 3 characters";
        if (value.length > 32) return "Username must be 32 characters or fewer";
        if (!/^[a-z0-9_-]+$/.test(value))
          return "Use only lowercase letters, numbers, hyphens, and underscores";
        return null;
      },
      display_name: (value) =>
        value.trim().length === 0 ? "Display name is required" : null,
      email: (value) =>
        value.trim().length > 0 && !/^[^@\s]+@[^@\s]+\.[^@\s]+$/.test(value.trim())
          ? "Enter a valid email address"
          : null,
    },
  });

  function handleClose() {
    form.reset();
    onClose();
  }

  function handleSubmit(values: CreateUserValues) {
    const email = values.email.trim();
    createUser.mutate(
      {
        body: {
          username: values.username,
          display_name: values.display_name,
          email: email || undefined,
        },
      },
      {
        onSuccess: (user) => {
          notifications.show({ color: "green", message: "User created" });
          handleClose();
          navigate(buildRoute.accessUserDetail(user.id));
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
          <Alert variant="light" color="blue">
            <Text size="sm">
              A user is just a container for grouping devices — it can&apos;t sign in on its
              own. To give someone access to PulseWeaver, open their profile and{" "}
              <Text component="span" fw={600}>
                Promote to admin
              </Text>
              , which sets a login password.
            </Text>
          </Alert>
          <TextInput
            label="Username"
            placeholder="e.g. jgarcia"
            description="3–32 characters · lowercase letters, numbers, hyphens and underscores"
            withAsterisk
            {...form.getInputProps("username")}
          />
          <TextInput
            label="Display name"
            placeholder="e.g. Juan Garcia"
            withAsterisk
            {...form.getInputProps("display_name")}
          />
          <TextInput
            label="Email"
            type="email"
            placeholder="e.g. juan@example.com"
            description="Optional"
            {...form.getInputProps("email")}
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
