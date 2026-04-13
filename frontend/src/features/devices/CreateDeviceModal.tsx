import { useState } from "react";
import { useForm, schemaResolver } from "@mantine/form";
import { z } from "zod";
import {
  ActionIcon,
  Button,
  Group,
  Modal,
  SegmentedControl,
  Select,
  SimpleGrid,
  Stack,
  Text,
  Textarea,
  TextInput,
  Tooltip,
  UnstyledButton,
} from "@mantine/core";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { notifications } from "@mantine/notifications";
import { useCreateDevice } from "@/features/devices/hooks/useCreateDevice";
import { useCurrentUser } from "@/features/auth/hooks/useCurrentUser";
import { useListUsers } from "@/features/auth/hooks/useListUsers";
import { IconPickerPopover } from "@/features/devices/IconPickerPopover";
import {
  DEVICE_TYPE_CONFIG,
  getDeviceIcon,
} from "@/features/devices/deviceTypeConfig";
import type { DeviceType } from "@/features/devices/deviceTypeConfig";
import { toApiError, toErrorMessage } from "@/lib/api-client";
import {
  updateDeviceMutation,
  getDevicesQueryKey,
} from "@/lib/api/@tanstack/react-query.gen";
import { UserRole } from "@/lib/api";
import { zCreateDeviceRequest, zUpdateDeviceRequest } from "@/lib/api/zod.gen";
import { IconX } from "@tabler/icons-react";

// ---------------------------------------------------------------------------
// Form schema
// ---------------------------------------------------------------------------

const formSchema = z.object({
  name: zCreateDeviceRequest.shape.name,
  device_type: zUpdateDeviceRequest.shape.device_type.unwrap(),
  description: z.string().max(200),
  icon: z.string().max(80),
});

type FormValues = z.infer<typeof formSchema>;

// ---------------------------------------------------------------------------
// CreateDeviceModal
// ---------------------------------------------------------------------------

interface CreateDeviceModalProps {
  opened: boolean;
  onClose: () => void;
}

export function CreateDeviceModal({ opened, onClose }: CreateDeviceModalProps) {
  const queryClient = useQueryClient();
  const { data: currentUser } = useCurrentUser();
  const isAdmin = currentUser?.role === UserRole.ADMIN;
  const { data: users } = useListUsers({ enabled: isAdmin });

  const ownerOptions = (users ?? []).map((u) => ({
    value: String(u.id),
    label: u.id === currentUser?.id ? `${u.display_name} (you)` : u.display_name,
  }));

  const [selectedOwner, setSelectedOwner] = useState<string | null>(null);
  const effectiveOwner = selectedOwner ?? (currentUser ? String(currentUser.id) : null);

  const [iconPickerOpen, setIconPickerOpen] = useState(false);

  const form = useForm<FormValues>({
    validate: schemaResolver(formSchema),
    initialValues: {
      name: "",
      device_type: "static" as DeviceType,
      description: "",
      icon: "",
    },
  });

  const { Icon: CurrentIcon, color: currentColor } = getDeviceIcon({
    device_type: form.values.device_type,
    icon: form.values.icon || null,
  });

  const segmentedData = (Object.keys(DEVICE_TYPE_CONFIG) as DeviceType[]).map(
    (v) => ({
      value: v,
      label: v.charAt(0).toUpperCase() + v.slice(1),
    }),
  );

  // Used only for patching optional profile fields after creation.
  const updateDevice = useMutation({
    ...updateDeviceMutation(),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: getDevicesQueryKey() });
    },
  });

  const createDevice = useCreateDevice({
    onSuccess: (data) => {
      const { device } = data;
      const patchBody: Record<string, unknown> = {};
      if (form.values.device_type !== "static")
        patchBody.device_type = form.values.device_type;
      const desc = form.values.description || null;
      if (desc) patchBody.description = desc;
      const icon = form.values.icon || null;
      if (icon) patchBody.icon = icon;

      if (Object.keys(patchBody).length > 0) {
        updateDevice.mutate(
          { path: { device_id: device.id }, body: patchBody },
          {
            onError: () =>
              notifications.show({
                color: "orange",
                message:
                  "Device created, but profile details could not be saved. You can update them in Settings.",
              }),
          },
        );
      }

      form.reset();
      setSelectedOwner(null);
      onClose();
    },
  });

  function handleClose() {
    form.reset();
    setSelectedOwner(null);
    onClose();
  }

  function onSubmit(values: FormValues) {
    const body: Record<string, unknown> = { name: values.name };
    if (isAdmin && effectiveOwner) {
      body.owner_id = Number(effectiveOwner);
    }
    createDevice.mutate(
      { body: body as Parameters<typeof createDevice.mutate>[0]["body"] },
      {
        onError: (err) => {
          const message =
            toApiError(err).status === 409
              ? "A device with this name already exists."
              : toErrorMessage(err);
          notifications.show({
            color: "red",
            title: "Error creating device",
            message,
          });
        },
      },
    );
  }

  const isPending = createDevice.isPending || updateDevice.isPending;

  return (
    <Modal
      opened={opened}
      onClose={handleClose}
      title="Create device"
      size="md"
    >
      <form onSubmit={form.onSubmit(onSubmit)}>
        <Stack gap="md">
            <TextInput
              label="Name"
              placeholder="e.g. Office Printer"
              data-autofocus
              {...form.getInputProps("name")}
            />

            {isAdmin && (
              <Select
                label="Owner"
                description="Admin only — defaults to you."
                data={ownerOptions}
                value={effectiveOwner}
                onChange={setSelectedOwner}
                searchable
              />
            )}

            <SimpleGrid cols={{ base: 1, sm: 2 }}>
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
                <Text size="sm" fw={500} mb={4}>
                  Icon
                </Text>
                <Group gap="xs" align="center">
                  <IconPickerPopover
                    opened={iconPickerOpen}
                    onClose={() => setIconPickerOpen(false)}
                    selectedIcon={form.values.icon}
                    onSelect={(name) => form.setFieldValue("icon", name)}
                    target={
                      <Tooltip label={form.values.icon || "Type default"} withArrow>
                        <UnstyledButton
                          onClick={() => setIconPickerOpen((o) => !o)}
                          style={{
                            display: "flex",
                            alignItems: "center",
                            justifyContent: "center",
                            width: 36,
                            height: 36,
                            borderRadius: "var(--mantine-radius-sm)",
                            border:
                              "1px solid var(--mantine-color-default-border)",
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
            </SimpleGrid>

            <div>
              <Textarea
                label="Description"
                placeholder="e.g. Juan's work MacBook, Living room Proxmox node"
                autosize
                maxRows={4}
                {...form.getInputProps("description")}
              />
              <Text size="xs" c="dimmed" ta="right" mt={2}>
                {form.values.description.length}/200
              </Text>
            </div>

            <Group justify="flex-end" gap="sm">
              <Button type="button" variant="outline" onClick={handleClose}>
                Cancel
              </Button>
              <Button type="submit" disabled={isPending} loading={isPending}>
                Create device
              </Button>
            </Group>
          </Stack>
        </form>
      </Modal>
  );
}
