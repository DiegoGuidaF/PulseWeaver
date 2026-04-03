import { useState } from "react";
import { useForm, schemaResolver } from "@mantine/form";
import {
  Button,
  Card,
  Group,
  Modal,
  Select,
  Stack,
  Text,
  TextInput,
  Title,
} from "@mantine/core";
import { notifications } from "@mantine/notifications";
import { useCreateDevice } from "@/features/devices/hooks/useCreateDevice";
import { useCurrentUser } from "@/features/auth/hooks/useCurrentUser";
import { useListUsers } from "@/features/auth/hooks/useListUsers";
import { toApiError, toErrorMessage } from "@/lib/api-client";
import type { CreateDeviceResponse } from "@/lib/api";
import { UserRole } from "@/lib/api";
import { zCreateDeviceRequest } from "@/lib/api/zod.gen";
import type { z } from "zod";

const formSchema = zCreateDeviceRequest;

export function CreateDeviceForm() {
  const { data: currentUser } = useCurrentUser();
  const isAdmin = currentUser?.role === UserRole.ADMIN;
  const { data: users } = useListUsers({ enabled: isAdmin });

  const ownerOptions = (users ?? []).map((u) => ({
    value: String(u.id),
    label: u.id === currentUser?.id ? `${u.display_name} (you)` : u.display_name,
  }));

  const [selectedOwner, setSelectedOwner] = useState<string | null>(null);

  // Resolve to current user's ID once loaded.
  const effectiveOwner = selectedOwner ?? (currentUser ? String(currentUser.id) : null);

  const form = useForm<z.infer<typeof formSchema>>({
    validate: schemaResolver(formSchema),
    initialValues: { name: "" },
  });

  const [createdResult, setCreatedResult] =
    useState<CreateDeviceResponse | null>(null);

  const mutation = useCreateDevice({
    onSuccess: (data) => {
      setCreatedResult(data);
      form.reset();
      setSelectedOwner(null);
    },
  });

  async function handleCopyApiKey() {
    if (!createdResult) return;
    if (!("clipboard" in navigator) || !navigator.clipboard?.writeText) {
      notifications.show({ message: "Copy to clipboard is not supported in this browser.", color: "red" });
      return;
    }
    try {
      await navigator.clipboard.writeText(createdResult.api_key);
      notifications.show({ message: "Copied to clipboard", color: "green" });
    } catch {
      notifications.show({ message: "Failed to copy API key", color: "red" });
    }
  }

  function onSubmit(values: z.infer<typeof formSchema>) {
    const body: z.infer<typeof formSchema> = { ...values };
    if (isAdmin && effectiveOwner) {
      body.owner_id = Number(effectiveOwner);
    }
    mutation.mutate(
      { body },
      {
        onError: (err) => {
          const message =
            toApiError(err).status === 409
              ? "A device with this name already exists."
              : toErrorMessage(err);
          notifications.show({ color: "red", title: "Error creating device", message });
        },
      },
    );
  }

  return (
    <Card withBorder>
      <Title order={4} mb="md">Add Device</Title>
      <form onSubmit={form.onSubmit(onSubmit)}>
        <Group align="flex-end" gap="md">
          <TextInput
            label="New Device Name"
            placeholder="e.g. Office Printer"
            style={{ flex: 1 }}
            {...form.getInputProps("name")}
          />
          {isAdmin && (
            <Select
              label="Owner"
              data={ownerOptions}
              value={effectiveOwner}
              onChange={setSelectedOwner}
              searchable
              style={{ minWidth: 180 }}
            />
          )}
          <Button type="submit" disabled={mutation.isPending}>
            {mutation.isPending ? "Creating..." : "Add Device"}
          </Button>
        </Group>
      </form>

      <Modal
        opened={createdResult !== null}
        onClose={() => setCreatedResult(null)}
        title="Device created — save your API key"
        closeOnClickOutside={false}
        closeOnEscape={false}
        withCloseButton={false}
      >
        <Stack gap="md">
          <Text size="sm" c="dimmed">
            This API key is shown only once. Copy it now and store it securely.
          </Text>
          {createdResult && (
            <>
              <Text size="sm">
                <Text component="span" fw={500}>Device:</Text>{" "}
                {createdResult.device.name}
              </Text>
              <Stack gap={8}>
                <Text size="sm" fw={500}>API key</Text>
                <Group gap="sm">
                  <TextInput
                    readOnly
                    value={createdResult.api_key}
                    ff="monospace"
                    style={{ flex: 1 }}
                  />
                  <Button type="button" variant="outline" onClick={handleCopyApiKey}>
                    Copy
                  </Button>
                </Group>
              </Stack>
              <Text size="xs" c="dimmed">
                You will not be able to see this full API key again. Make sure
                you have stored it securely.
              </Text>
            </>
          )}
          <Group justify="flex-end">
            <Button type="button" onClick={() => setCreatedResult(null)}>
              I&apos;ve saved it
            </Button>
          </Group>
        </Stack>
      </Modal>
    </Card>
  );
}
