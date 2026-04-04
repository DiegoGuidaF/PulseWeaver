import { useEffect, useState } from "react";
import { useForm, schemaResolver } from "@mantine/form";
import { z } from "zod";
import {
  ActionIcon,
  Button,
  Card,
  Group,
  Popover,
  SegmentedControl,
  SimpleGrid,
  Stack,
  Text,
  Textarea,
  TextInput,
  UnstyledButton,
} from "@mantine/core";
import { IconX } from "@tabler/icons-react";
import { notifications } from "@mantine/notifications";
import { toErrorMessage } from "@/lib/api-client";
import { useUpdateDevice } from "@/features/devices/hooks/useUpdateDevice";
import {
  DEVICE_TYPE_CONFIG,
  ICON_PICKER_OPTIONS,
  getDeviceIcon,
} from "@/features/devices/deviceTypeConfig";
import type { DeviceType } from "@/features/devices/deviceTypeConfig";
import type { DeviceTypeItem } from "@/lib/api";
import { zUpdateDeviceRequest } from "@/lib/api/zod.gen";

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export interface DeviceForProfile {
  name: string;
  device_type: DeviceType;
  description?: string | null;
  icon?: string | null;
}

// ---------------------------------------------------------------------------
// Form schema
// ---------------------------------------------------------------------------

// Derived from the generated zUpdateDeviceRequest schema so constraints
// stay in sync with the OpenAPI spec automatically on `make api`.
//
// name / device_type: .unwrap() strips .optional() — the form always
//   has all fields populated (unlike the PATCH request which allows partials).
// description / icon: the API accepts null/undefined, but the form holds plain
//   strings (empty string = null sentinel, transformed on submit).
const profileFormSchema = z.object({
  name: zUpdateDeviceRequest.shape.name.unwrap(),
  device_type: zUpdateDeviceRequest.shape.device_type.unwrap(),
  description: z.string().max(200),
  icon: z.string().max(80),
});

type ProfileFormValues = z.infer<typeof profileFormSchema>;

function deviceToFormValues(d: DeviceForProfile): ProfileFormValues {
  return {
    name: d.name,
    device_type: d.device_type,
    description: d.description ?? "",
    icon: d.icon ?? "",
  };
}

// ---------------------------------------------------------------------------
// Icon picker popover
// ---------------------------------------------------------------------------

interface IconPickerPopoverProps {
  opened: boolean;
  onClose: () => void;
  target: React.ReactNode;
  selectedIcon: string;
  onSelect: (name: string) => void;
}

function IconPickerPopover({
  opened,
  onClose,
  target,
  selectedIcon,
  onSelect,
}: IconPickerPopoverProps) {
  return (
    <Popover
      opened={opened}
      onClose={onClose}
      position="bottom-start"
      withinPortal
      shadow="md"
    >
      <Popover.Target>{target}</Popover.Target>
      <Popover.Dropdown>
        <SimpleGrid cols={5} spacing={4}>
          {ICON_PICKER_OPTIONS.map(({ name, icon: Icon }) => (
            <ActionIcon
              key={name}
              variant={selectedIcon === name ? "filled" : "subtle"}
              size="lg"
              aria-label={name}
              onClick={() => {
                onSelect(name);
                onClose();
              }}
            >
              <Icon size={18} />
            </ActionIcon>
          ))}
        </SimpleGrid>
      </Popover.Dropdown>
    </Popover>
  );
}

// ---------------------------------------------------------------------------
// DeviceProfileCard
// ---------------------------------------------------------------------------

export interface DeviceProfileCardProps {
  deviceId: number;
  device: DeviceForProfile;
  deviceTypes: DeviceTypeItem[];
}

export function DeviceProfileCard({
  deviceId,
  device,
  deviceTypes,
}: DeviceProfileCardProps) {
  const updateDevice = useUpdateDevice(deviceId);
  const [iconPickerOpen, setIconPickerOpen] = useState(false);

  const form = useForm<ProfileFormValues>({
    validate: schemaResolver(profileFormSchema),
    initialValues: deviceToFormValues(device),
  });

  // Sync with latest server state, but never overwrite an in-progress edit.
  useEffect(() => {
    if (form.isDirty()) return;
    form.setValues(deviceToFormValues(device));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [device]);

  const segmentedData =
    deviceTypes.length > 0
      ? deviceTypes.map((t) => ({ value: t.value, label: t.label }))
      : (Object.keys(DEVICE_TYPE_CONFIG) as DeviceType[]).map((v) => ({
          value: v,
          label: v.charAt(0).toUpperCase() + v.slice(1),
        }));

  const { Icon: CurrentIcon, color: currentColor } = getDeviceIcon({
    device_type: form.values.device_type,
    icon: form.values.icon || null,
  });

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
    if (values.device_type !== device.device_type)
      body.device_type = values.device_type;
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
          <Group justify="space-between" align="center">
            <Text fw={500}>
              Device profile{" "}
              {isDirty && (
                <Text component="span" c="yellow.5" aria-label="unsaved changes">
                  •
                </Text>
              )}
            </Text>
          </Group>

          <TextInput
            label="Name"
            placeholder="My device"
            {...form.getInputProps("name")}
          />

          <div>
            <Text size="sm" fw={500} mb={4}>
              Type
            </Text>
            <SegmentedControl
              data={segmentedData}
              value={form.values.device_type}
              onChange={(val) =>
                form.setFieldValue("device_type", val as DeviceType)
              }
            />
          </div>

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

          <div>
            <Text size="sm" fw={500} mb={4}>
              Icon
            </Text>
            <Group gap="sm" align="center">
              <IconPickerPopover
                opened={iconPickerOpen}
                onClose={() => setIconPickerOpen(false)}
                selectedIcon={form.values.icon}
                onSelect={(name) => form.setFieldValue("icon", name)}
                target={
                  <UnstyledButton
                    onClick={() => setIconPickerOpen((o) => !o)}
                    style={{
                      display: "flex",
                      alignItems: "center",
                      gap: 8,
                      padding: "6px 10px",
                      borderRadius: "var(--mantine-radius-sm)",
                      border: "1px solid var(--mantine-color-default-border)",
                      cursor: "pointer",
                    }}
                  >
                    <CurrentIcon
                      size={20}
                      style={{
                        color:
                          currentColor === "dimmed"
                            ? "var(--mantine-color-dimmed)"
                            : `var(--mantine-color-${currentColor}-filled)`,
                      }}
                    />
                    <Text size="sm" c={form.values.icon ? undefined : "dimmed"}>
                      {form.values.icon || "Type default"}
                    </Text>
                  </UnstyledButton>
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

          <Group justify="flex-end" gap="sm">
            {isDirty && (
              <Button
                type="button"
                variant="subtle"
                size="sm"
                onClick={handleReset}
              >
                Reset
              </Button>
            )}
            <Button
              type="submit"
              size="sm"
              disabled={!isDirty || updateDevice.isPending}
              loading={updateDevice.isPending}
            >
              Save
            </Button>
          </Group>
        </Stack>
      </form>
    </Card>
  );
}
