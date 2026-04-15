import { useState } from "react";
import { useForm, schemaResolver } from "@mantine/form";
import {
  Button,
  Collapse,
  Fieldset,
  SegmentedControl,
  Select,
  Stack,
  Switch,
  Text,
  TextInput,
} from "@mantine/core";
import { notifications } from "@mantine/notifications";
import { useCreateRegistration } from "./hooks/useCreateRegistration";
import { zCreateRegistrationRequest } from "@/lib/api/zod.gen";
import type { PendingRegistration } from "@/lib/api";
import { toErrorMessage } from "@/lib/api-client";

interface InviteCreationFormProps {
  onSuccess: (registration: PendingRegistration) => void;
}

type FormValues = {
  device_name: string;
  heartbeat_server_url: string;
  interval_seconds: number;
  biometric_enabled: boolean;
  biometric_user_can_toggle: boolean;
  expires_in_hours: 1 | 24 | 48 | 168;
};

export function InviteCreationForm({ onSuccess }: InviteCreationFormProps) {
  const mutation = useCreateRegistration();
  const [showBiometric, setShowBiometric] = useState(false);

  const form = useForm<FormValues>({
    validate: schemaResolver(zCreateRegistrationRequest),
    initialValues: {
      device_name: "",
      heartbeat_server_url: window.location.origin,
      interval_seconds: 900,
      biometric_enabled: false,
      biometric_user_can_toggle: true,
      expires_in_hours: 24,
    },
  });

  function onSubmit(values: FormValues) {
    mutation.mutate(
      { body: values },
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
              label="Heartbeat server URL"
              {...form.getInputProps("heartbeat_server_url")}
            />
            <Switch
              label="Configure biometric settings"
              checked={showBiometric}
              onChange={(e) => setShowBiometric(e.currentTarget.checked)}
            />
            <Collapse expanded={showBiometric}>
              <Stack gap="sm" pt="xs">
                <Switch
                  label="Enable biometrics"
                  {...form.getInputProps("biometric_enabled", {
                    type: "checkbox",
                  })}
                />
                {form.values.biometric_enabled && (
                  <Switch
                    label="User can toggle biometrics"
                    {...form.getInputProps("biometric_user_can_toggle", {
                      type: "checkbox",
                    })}
                  />
                )}
              </Stack>
            </Collapse>
          </Stack>
        </Fieldset>

        <Button type="submit" loading={mutation.isPending}>
          Create invite →
        </Button>
      </Stack>
    </form>
  );
}
