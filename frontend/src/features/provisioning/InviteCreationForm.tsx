import { schemaResolver, useForm } from "@mantine/form";
import { Button, Fieldset, Group, SegmentedControl, Select, Stack, Switch, Text, TextInput } from "@mantine/core";
import { notifications } from "@mantine/notifications";
import { useCreateRegistration } from "./hooks/useCreateRegistration";
import { zCreateRegistrationRequest } from "@/lib/api/zod.gen";
import type { z } from "zod";
import { type PendingRegistration } from "@/lib/api";
import { toErrorMessage } from "@/lib/api-client";
import { useListUsers } from "@/features/auth/hooks/useListUsers";
import { useCurrentUser } from "@/features/auth/hooks/useCurrentUser";
import { useState } from "react";

type CreateRegistrationValues = z.infer<typeof zCreateRegistrationRequest>;

interface InviteCreationFormProps {
  onSuccess: (registration: PendingRegistration) => void;
  onCancel?: () => void;
}

export function InviteCreationForm({ onSuccess, onCancel }: InviteCreationFormProps) {
  const mutation = useCreateRegistration();
  const { data: currentUser } = useCurrentUser();
  const { data: users } = useListUsers();

  const ownerOptions = (users ?? []).map((u) => ({
    value: String(u.id),
    label: u.id === currentUser?.id ? `${u.display_name} (you)` : u.display_name,
  }));

  const [selectedOwner, setSelectedOwner] = useState<string | null>(
    currentUser ? String(currentUser.id) : null,
  );

  const form = useForm<CreateRegistrationValues>({
    validate: schemaResolver(zCreateRegistrationRequest),
    initialValues: {
      device_name: "",
      owner_id: BigInt(1),
      heartbeat_server_url: window.location.origin,
      interval_seconds: 900,
      app_biometric_enabled: false,
      app_settings_locked: false,
      expires_in_hours: 24,
    },
  });

  function onSubmit(values: CreateRegistrationValues) {
    mutation.mutate(
      { body: { ...values, owner_id: Number(selectedOwner ?? currentUser?.id) } },
      {
        onSuccess: (data) => onSuccess(data),
        onError: (err) =>
          notifications.show({
            color: "red",
            title: "Failed to create invite",
            message: toErrorMessage(err),
          }),
      },
    );
  }

  return (
    <form onSubmit={form.onSubmit(onSubmit)}>
      <Stack>
        <Fieldset legend="Invitation">
          <Stack gap="sm">
            <TextInput
              label="Device name"
              placeholder="e.g. Office Laptop"
              {...form.getInputProps("device_name")}
            />
            <Select
              label="Owner"
              description="User who will own this device."
              data={ownerOptions}
              value={selectedOwner}
              onChange={setSelectedOwner}
              searchable
            />
            <Select
              label="Expires in"
              data={[
                { value: "1", label: "1 hour" },
                { value: "24", label: "24 hours" },
                { value: "48", label: "48 hours" },
                { value: "168", label: "7 days" },
              ]}
              value={String(form.values.expires_in_hours)}
              onChange={(val) => {
                if (val)
                  form.setFieldValue(
                    "expires_in_hours",
                    Number(val) as 1 | 24 | 48 | 168,
                  );
              }}
            />
          </Stack>
        </Fieldset>

        <Fieldset legend="App configuration">
          <Stack gap="sm">
            <Text size="sm" c="dimmed">
              When the user registers, these settings will be automatically
              applied to their app.
            </Text>
            <div>
              <Text size="sm" fw={500} mb="xs">
                Heartbeat interval
              </Text>
              <SegmentedControl
                data={[
                  { value: "300", label: "5 min" },
                  { value: "900", label: "15 min" },
                  { value: "1800", label: "30 min" },
                  { value: "3600", label: "1 hour" },
                ]}
                value={String(form.values.interval_seconds)}
                onChange={(val) =>
                  form.setFieldValue("interval_seconds", Number(val))
                }
              />
            </div>
            <TextInput
              label="External server URL (for heartbeat and register endpoints)"
              {...form.getInputProps("heartbeat_server_url")}
            />
            <Switch
              label="Enable biometric unlock"
              {...form.getInputProps("app_biometric_enabled", {
                type: "checkbox",
              })}
            />
            <Switch
              label="Lock all app settings on device"
              description="Prevents the user from changing any app settings."
              {...form.getInputProps("app_settings_locked", {
                type: "checkbox",
              })}
            />
          </Stack>
        </Fieldset>

        <Group justify="flex-end" gap="sm">
          {onCancel && (
            <Button type="button" variant="outline" onClick={onCancel}>
              Cancel
            </Button>
          )}
          <Button type="submit" loading={mutation.isPending}>
            Create invite →
          </Button>
        </Group>
      </Stack>
    </form>
  );
}
