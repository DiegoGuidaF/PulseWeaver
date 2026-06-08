import { useEffect, useState } from "react";
import { useForm, schemaResolver } from "@mantine/form";
import { z } from "zod";
import {
  ActionIcon,
  Button,
  Card,
  Group,
  Stack,
  Text,
  Textarea,
  TextInput,
  Tooltip,
  UnstyledButton,
} from "@mantine/core";
import { IconPickerPopover } from "@/features/devices/IconPickerPopover";
import { IconX } from "@tabler/icons-react";
import { notifications } from "@mantine/notifications";
import { toErrorMessage } from "@/lib/api-client";
import { useUpdateDevice } from "@/features/devices/hooks/useUpdateDevice";
import { resolveDeviceIcon } from "@/features/devices/deviceTypeConfig";
import { zUpdateDeviceRequest } from "@/lib/api/zod.gen";

export interface DeviceForProfile {
  name: string;
  description?: string | null;
  icon?: string | null;
}

const profileFormSchema = z.object({
  name: zUpdateDeviceRequest.shape.name.unwrap(),
  description: z.string().max(200),
  icon: z.string().max(80),
});

type ProfileFormValues = z.infer<typeof profileFormSchema>;

function deviceToFormValues(d: DeviceForProfile): ProfileFormValues {
  return {
    name: d.name,
    description: d.description ?? "",
    icon: d.icon ?? "",
  };
}

export interface DeviceProfileCardProps {
  deviceId: number;
  device: DeviceForProfile;
}

export function DeviceProfileCard({
  deviceId,
  device,
}: DeviceProfileCardProps) {
  const updateDevice = useUpdateDevice();
  const [iconPickerOpen, setIconPickerOpen] = useState(false);

  const form = useForm<ProfileFormValues>({
    validateInputOnBlur: true,
    validate: schemaResolver(profileFormSchema),
    initialValues: deviceToFormValues(device),
  });

  // Sync with latest server state, but never overwrite an in-progress edit.
  useEffect(() => {
    if (form.isDirty()) return;
    form.setValues(deviceToFormValues(device));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [device]);

  const renderIcon = resolveDeviceIcon(form.values.icon || null);

  const isDirty = form.isDirty();
  const descLen = form.values.description.length;

  function handleReset() {
    const vals = deviceToFormValues(device);
    form.setValues(vals);
    form.resetDirty(vals);
  }

  function handleSubmit(values: ProfileFormValues) {
    const body: Record<string, unknown> = {};
    if (values.name !== device.name) body.name = values.name;
    const newDesc = values.description || null;
    if (newDesc !== (device.description ?? null)) body.description = newDesc;
    const newIcon = values.icon || null;
    if (newIcon !== (device.icon ?? null)) body.icon = newIcon;

    updateDevice.mutate(
      { path: { device_id: deviceId }, body },
      {
        onSuccess: () => {
          form.resetDirty(values);
          notifications.show({ color: "green", message: "Device profile saved" });
        },
        onError: (err) => {
          const status =
            err && typeof err === "object" && "status" in err
              ? (err as { status: unknown }).status
              : undefined;
          if (status === 409) {
            form.setFieldError("name", "Name already in use");
          } else {
            notifications.show({ color: "red", message: toErrorMessage(err) });
          }
        },
      },
    );
  }

  return (
    <Card withBorder>
      <form onSubmit={form.onSubmit(handleSubmit)}>
        <Stack gap="md">
          <TextInput
            label="Name"
            placeholder="My device"
            withAsterisk
            {...form.getInputProps("name")}
          />

          <Group gap="lg" align="flex-end">
            <div>
              <Text size="sm" fw={500} mb={4}>
                Icon
              </Text>
              <Group gap="xs" align="center">
                <IconPickerPopover
                  opened={iconPickerOpen}
                  onClose={() => setIconPickerOpen(false)}
                  selectedIcon={form.values.icon}
                  onSelect={(name) => form.setFieldValue("icon", name)}
                  deviceName={form.values.name}
                  target={
                    <Tooltip
                      label={form.values.icon || "Type default"}
                      withArrow
                    >
                      <UnstyledButton
                        onClick={() => setIconPickerOpen((o) => !o)}
                        style={{
                          display: "flex",
                          alignItems: "center",
                          justifyContent: "center",
                          width: 36,
                          height: 36,
                          borderRadius: "var(--mantine-radius-sm)",
                          border: "1px solid var(--mantine-color-default-border)",
                          cursor: "pointer",
                        }}
                      >
                        {renderIcon({ size: 20 })}
                      </UnstyledButton>
                    </Tooltip>
                  }
                />
                {form.values.icon && (
                  <ActionIcon
                    variant="subtle"
                    color="dimmed"
                    size="sm"
                    aria-label="Clear icon override"
                    onClick={() => form.setFieldValue("icon", "")}
                  >
                    <IconX size={14} />
                  </ActionIcon>
                )}
              </Group>
            </div>
          </Group>

          <div>
            <Textarea
              label="Description"
              placeholder="e.g. Juan's work MacBook, Living room Proxmox node"
              autosize
              maxRows={4}
              {...form.getInputProps("description")}
            />
            <Text size="xs" c="dimmed" ta="right" mt={2}>
              {descLen}/200
            </Text>
          </div>

          {isDirty && (
            <Group justify="space-between" align="center">
              <Text size="xs" c="yellow.5">Unsaved changes</Text>
              <Group gap="sm">
                <Button
                  type="button"
                  variant="subtle"
                  size="sm"
                  onClick={handleReset}
                >
                  Reset
                </Button>
                <Button
                  type="submit"
                  size="sm"
                  loading={updateDevice.isPending}
                >
                  Save
                </Button>
              </Group>
            </Group>
          )}
        </Stack>
      </form>
    </Card>
  );
}
