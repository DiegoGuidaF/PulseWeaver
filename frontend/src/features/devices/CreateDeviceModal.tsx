import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { schemaResolver, useForm } from "@mantine/form";
import { z } from "zod";
import {
  ActionIcon,
  Anchor,
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
import type { DeviceType } from "@/features/devices/deviceTypeConfig";
import { DEVICE_TYPE_CONFIG, getDeviceIcon, suggestIcon } from "@/features/devices/deviceTypeConfig";
import { toApiError, toErrorMessage } from "@/lib/api-client";
import { buildRoute } from "@/lib/routes";
import { getDevicesQueryKey, updateDeviceMutation } from "@/lib/api/@tanstack/react-query.gen";
import { zCreateDeviceRequest, zUpdateDeviceRequest } from "@/lib/api/zod.gen";
import { IconX } from "@tabler/icons-react";

const formSchema = z.object({
  name: zCreateDeviceRequest.shape.name,
  device_type: zUpdateDeviceRequest.shape.device_type.unwrap(),
  description: z.string().max(200),
  icon: z.string().max(80),
});

type FormValues = z.infer<typeof formSchema>;
interface CreateDeviceModalProps {
  opened: boolean;
  onClose: () => void;
  defaultOwnerId?: number | null;
}

export function CreateDeviceModal({ opened, onClose, defaultOwnerId }: CreateDeviceModalProps) {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { data: currentUser } = useCurrentUser();
  const { data: users } = useListUsers({ enabled: currentUser != null });

  const ownerOptions = (users ?? []).map((u) => ({
    value: u.id,
    label: u.id === currentUser?.id ? `${u.display_name} (you)` : u.display_name,
  }));

  const [selectedOwnerId, setSelectedOwnerId] = useState<number | null>(defaultOwnerId ?? null);
  const effectiveOwner = selectedOwnerId ?? (currentUser ? Number(currentUser.id) : null);
  const [ownerEditing, setOwnerEditing] = useState(!defaultOwnerId);

  const effectiveOwnerName = (users ?? []).find(
    (u) => Number(u.id) === effectiveOwner,
  )?.display_name ?? null;

  const [iconPickerOpen, setIconPickerOpen] = useState(false);
  const [iconAutoSuggested, setIconAutoSuggested] = useState(true);

  const form = useForm<FormValues>({
    validate: schemaResolver(formSchema),
    initialValues: {
      name: "",
      device_type: "static" as DeviceType,
      description: "",
      icon: "",
    },
  });

  const { setFieldValue } = form;
  const nameValue = form.values.name;
  useEffect(() => {
    if (!iconAutoSuggested) return;
    setFieldValue("icon", suggestIcon(nameValue));
  }, [nameValue, iconAutoSuggested, setFieldValue]);

  const renderIcon = getDeviceIcon({
    device_type: form.values.device_type,
    icon: form.values.icon || null,
  });
  const isIconAutoSuggested = iconAutoSuggested && !!form.values.icon;

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

  function resetForm() {
    form.reset();
    setSelectedOwnerId(defaultOwnerId ?? null);
    setOwnerEditing(!defaultOwnerId);
    setIconAutoSuggested(true);
  }

  const createDevice = useCreateDevice({
    onSuccess: (data) => {
      const patchBody: Record<string, unknown> = {};
      if (form.values.device_type !== "static")
        patchBody.device_type = form.values.device_type;
      const desc = form.values.description || null;
      if (desc) patchBody.description = desc;
      const icon = form.values.icon || null;
      if (icon) patchBody.icon = icon;

      if (Object.keys(patchBody).length > 0) {
        updateDevice.mutate(
          { path: { device_id: data.id }, body: patchBody },
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

      resetForm();
      onClose();
      if (effectiveOwner) {
        navigate(
          `${buildRoute.userDeviceWorkspace(effectiveOwner)}?device=${data.id}&tab=addresses`,
        );
      }
    },
  });

  function handleClose() {
    resetForm();
    onClose();
  }

  function onSubmit(values: FormValues) {
    const body: Record<string, unknown> = { name: values.name };
    body.owner_id = effectiveOwner;
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

            {ownerEditing ? (
              <Select
                label="Owner"
                data={ownerOptions}
                value={effectiveOwner}
                onChange={(val) => {
                  setSelectedOwnerId(val as unknown as number | null);
                  if (val) setOwnerEditing(false);
                }}
                searchable
                autoFocus
              />
            ) : (
              <div>
                <Text size="sm" fw={500} mb={4}>Owner</Text>
                <Group gap="xs" align="center">
                  <Text size="sm">{effectiveOwnerName ?? "—"}</Text>
                  <Anchor
                    component="button"
                    type="button"
                    size="xs"
                    c="dimmed"
                    onClick={() => setOwnerEditing(true)}
                  >
                    change
                  </Anchor>
                </Group>
              </div>
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
                    onSelect={(name) => {
                      setIconAutoSuggested(false);
                      form.setFieldValue("icon", name);
                    }}
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
                      onClick={() => {
                        setIconAutoSuggested(true);
                        form.setFieldValue("icon", suggestIcon(form.values.name));
                      }}
                    >
                      <IconX size={14} />
                    </ActionIcon>
                  )}
                </Group>
                {isIconAutoSuggested && (
                  <Text size="xs" c="dimmed" mt={4}>
                    suggested from name
                  </Text>
                )}
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
