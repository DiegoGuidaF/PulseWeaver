import { schemaResolver, useForm } from "@mantine/form";
import {
  Button,
  Fieldset,
  Group,
  SegmentedControl,
  Stack,
  Switch,
  Text,
  TextInput,
} from "@mantine/core";
import { notifications } from "@mantine/notifications";
import { zCreatePairingRequest } from "@/lib/api/zod.gen";
import type { z } from "zod";
import type { DevicePairing } from "@/lib/api";
import { toErrorMessage } from "@/lib/api-client";
import { useCreateDevicePairing } from "./hooks/useCreateDevicePairing";

const LS_SERVER_URL_KEY = "pw.pair.serverUrl";

type FormValues = z.infer<typeof zCreatePairingRequest>;

interface Props {
  deviceId: number;
  onSuccess: (pairing: DevicePairing) => void;
  onCancel: () => void;
}

export function PairingCreationForm({ deviceId, onSuccess, onCancel }: Props) {
  const mutation = useCreateDevicePairing(deviceId);

  const form = useForm<FormValues>({
    validate: schemaResolver(zCreatePairingRequest),
    initialValues: {
      heartbeat_server_url: localStorage.getItem(LS_SERVER_URL_KEY) ?? window.location.origin,
      interval_seconds: 1800,
      app_biometric_enabled: true,
      app_settings_locked: false,
      expires_in_hours: 24,
    },
  });

  function onSubmit(values: FormValues) {
    mutation.mutate(
      { path: { id: deviceId }, body: values },
      {
        onSuccess: (data) => {
          localStorage.setItem(LS_SERVER_URL_KEY, values.heartbeat_server_url);
          onSuccess(data);
        },
        onError: (err) =>
          notifications.show({
            color: "red",
            title: "Failed to create pairing code",
            message: toErrorMessage(err),
          }),
      },
    );
  }

  return (
    <form onSubmit={form.onSubmit(onSubmit)}>
      <Stack>
        <TextInput
          label="Heartbeat server URL"
          description="URL the companion app uses to send heartbeats. Defaults to this browser's origin."
          required
          {...form.getInputProps("heartbeat_server_url")}
        />

        <div>
          <Text size="sm" fw={500} mb={4}>
            Heartbeat interval
          </Text>
          <SegmentedControl
            data={[
              { value: "900", label: "15 min" },
              { value: "1800", label: "30 min" },
              { value: "3600", label: "1 hour" },
            ]}
            value={String(form.values.interval_seconds)}
            onChange={(val) => form.setFieldValue("interval_seconds", Number(val))}
          />
        </div>

        <Fieldset legend="Companion app initial config">
          <Stack gap="sm">
            <Text size="sm" c="dimmed">
              These settings are pushed to the app at claim time and cannot be changed afterwards.
            </Text>
            <Switch
              label="Biometric unlock"
              description="FaceID / fingerprint required to open the app"
              {...form.getInputProps("app_biometric_enabled", { type: "checkbox" })}
            />
            <Switch
              label="Lock app settings"
              description="User sees a read-only settings screen in the companion"
              {...form.getInputProps("app_settings_locked", { type: "checkbox" })}
            />
          </Stack>
        </Fieldset>

        <div>
          <Text size="sm" fw={500} mb={4}>
            Code expires
          </Text>
          <SegmentedControl
            data={[
              { value: "1", label: "1 hour" },
              { value: "24", label: "24 hours" },
              { value: "48", label: "48 hours" },
              { value: "168", label: "7 days" },
            ]}
            value={String(form.values.expires_in_hours)}
            onChange={(val) =>
              form.setFieldValue("expires_in_hours", Number(val) as 1 | 24 | 48 | 168)
            }
          />
          <Text size="xs" c="dimmed" mt={4}>
            24h is the default — long enough to wait for a user to be at their desk.
          </Text>
        </div>

        <Group justify="flex-end" gap="sm">
          <Button type="button" variant="outline" onClick={onCancel}>
            Cancel
          </Button>
          <Button type="submit" loading={mutation.isPending}>
            Generate code →
          </Button>
        </Group>
      </Stack>
    </form>
  );
}
