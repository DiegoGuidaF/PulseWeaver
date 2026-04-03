import { useEffect, useState } from "react";
import { useForm, schemaResolver } from "@mantine/form";
import { z } from "zod";
import {
  ActionIcon,
  Badge,
  Button,
  Card,
  Group,
  Modal,
  NativeSelect,
  Popover,
  SegmentedControl,
  Select,
  SimpleGrid,
  Skeleton,
  Stack,
  Text,
  Textarea,
  TextInput,
  Title,
  UnstyledButton,
} from "@mantine/core";
import { IconX } from "@tabler/icons-react";
import { notifications } from "@mantine/notifications";
import { toErrorMessage } from "@/lib/api-client";
import { useCurrentUser } from "@/features/auth/hooks/useCurrentUser";
import { useListUsers } from "@/features/auth/hooks/useListUsers";
import { UserRole } from "@/lib/api";
import { useDeviceAddressLeaseRule } from "@/features/devices/hooks/useDeviceAddressLeaseRule";
import { usePutDeviceAddressLeaseRule } from "@/features/devices/hooks/usePutDeviceAddressLeaseRule";
import { useDisableDeviceAddressLeaseRule } from "@/features/devices/hooks/useDisableDeviceAddressLeaseRule";
import { useRegenerateApiKey } from "@/features/devices/hooks/useRegenerateApiKey";
import { useMaxActiveAddressesRule } from "@/features/devices/hooks/useMaxActiveAddressesRule";
import { usePutMaxActiveAddressesRule } from "@/features/devices/hooks/usePutMaxActiveAddressesRule";
import { useDisableMaxActiveAddressesRule } from "@/features/devices/hooks/useDisableMaxActiveAddressesRule";
import { useDeviceTypes } from "@/features/devices/hooks/useDeviceTypes";
import { useUpdateDevice } from "@/features/devices/hooks/useUpdateDevice";
import {
  DEVICE_TYPE_CONFIG,
  ICON_PICKER_OPTIONS,
  getDeviceIcon,
} from "@/features/devices/deviceTypeConfig";
import type { DeviceType } from "@/features/devices/deviceTypeConfig";
import type { DeviceTypeItem } from "@/lib/api";

// ---------------------------------------------------------------------------
// TTL helpers (unchanged)
// ---------------------------------------------------------------------------

const TTL_UNITS = ["seconds", "minutes", "hours", "days"] as const;
const SECONDS_PER_MINUTE = 60;
const SECONDS_PER_HOUR = 3600;
const SECONDS_PER_DAY = 86400;

type TtlUnit = (typeof TTL_UNITS)[number];

function toSeconds(value: number, unit: TtlUnit): number {
  switch (unit) {
    case "seconds":
      return value;
    case "minutes":
      return value * SECONDS_PER_MINUTE;
    case "hours":
      return value * SECONDS_PER_HOUR;
    case "days":
      return value * SECONDS_PER_DAY;
    default: {
      const _exhaustive: never = unit;
      return _exhaustive;
    }
  }
}

function fromSeconds(ttlSeconds: number): { value: string; unit: TtlUnit } {
  if (ttlSeconds % SECONDS_PER_DAY === 0) {
    return { value: String(ttlSeconds / SECONDS_PER_DAY), unit: "days" };
  }
  if (ttlSeconds % SECONDS_PER_HOUR === 0) {
    return { value: String(ttlSeconds / SECONDS_PER_HOUR), unit: "hours" };
  }
  if (ttlSeconds % SECONDS_PER_MINUTE === 0) {
    return { value: String(ttlSeconds / SECONDS_PER_MINUTE), unit: "minutes" };
  }
  return { value: String(ttlSeconds), unit: "seconds" };
}

function formatTtlLabel(ttlSeconds: number): string {
  if (ttlSeconds % SECONDS_PER_DAY === 0) {
    const days = ttlSeconds / SECONDS_PER_DAY;
    return days === 1 ? "1 day" : `${days} days`;
  }
  if (ttlSeconds % SECONDS_PER_MINUTE === 0) {
    const minutes = ttlSeconds / SECONDS_PER_MINUTE;
    if (minutes % 60 === 0) {
      const hours = minutes / 60;
      return hours === 1 ? "1 hour" : `${hours} hours`;
    }
    return minutes === 1 ? "1 minute" : `${minutes} minutes`;
  }
  return ttlSeconds === 1 ? "1 second" : `${ttlSeconds} seconds`;
}

// ---------------------------------------------------------------------------
// Device profile card
// ---------------------------------------------------------------------------

// Constraints mirror zUpdateDeviceRequest (generated from OpenAPI spec).
// All form fields are strings; empty string is the null sentinel for nullable
// fields (description, icon) — transformed on submit.
const profileFormSchema = z.object({
  name: z.string().min(1, "Name is required").max(50),
  device_type: z.enum(["static", "mobile"]),
  description: z.string().max(200),
  icon: z.string().max(80),
});

type ProfileFormValues = z.infer<typeof profileFormSchema>;

interface DeviceForProfile {
  name: string;
  device_type: DeviceType;
  description?: string | null;
  icon?: string | null;
}

function deviceToFormValues(d: DeviceForProfile): ProfileFormValues {
  return {
    name: d.name,
    device_type: d.device_type,
    description: d.description ?? "",
    icon: d.icon ?? "",
  };
}

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

interface DeviceProfileCardProps {
  deviceId: number;
  device: DeviceForProfile;
  deviceTypes: DeviceTypeItem[];
}

function DeviceProfileCard({
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

// ---------------------------------------------------------------------------
// Ownership card
// ---------------------------------------------------------------------------

interface DeviceOwnershipCardProps {
  deviceId: number;
  ownerId?: number;
  ownerName?: string;
}

function DeviceOwnershipCard({
  deviceId,
  ownerId,
  ownerName,
}: DeviceOwnershipCardProps) {
  const { data: currentUser } = useCurrentUser();
  const isAdmin = currentUser?.role === UserRole.ADMIN;

  const { data: users, isLoading: usersLoading } = useListUsers({
    enabled: isAdmin,
  });

  const updateDevice = useUpdateDevice(deviceId);
  const [selectedOwner, setSelectedOwner] = useState(
    ownerId != null ? String(ownerId) : "",
  );

  const ownerDirty =
    selectedOwner !== (ownerId != null ? String(ownerId) : "");

  function handleOwnerSave() {
    if (!selectedOwner) return;
    updateDevice.mutate(
      {
        path: { device_id: deviceId },
        body: { owner_id: Number(selectedOwner) },
      },
      {
        onSuccess: () =>
          notifications.show({
            color: "green",
            message: "Device ownership updated",
          }),
        onError: (err) => {
          const status =
            err && typeof err === "object" && "status" in err
              ? (err as { status: unknown }).status
              : undefined;
          notifications.show({
            color: "red",
            message:
              status === 403
                ? "Admin permission required to reassign ownership"
                : toErrorMessage(err),
          });
        },
      },
    );
  }

  const selectData =
    users?.map((u) => ({ value: String(u.id), label: u.display_name })) ?? [];

  return (
    <Card withBorder>
      <Stack gap="md">
        <Text fw={500}>Ownership</Text>
        {!isAdmin ? (
          <Group gap="xs">
            <Text size="sm" c="dimmed">
              Owned by
            </Text>
            <Text size="sm">{ownerName ?? "—"}</Text>
          </Group>
        ) : usersLoading ? (
          <Skeleton height={36} width={240} />
        ) : (
          <>
            <Select
              label="Owner"
              data={selectData}
              value={selectedOwner}
              onChange={(val) => setSelectedOwner(val ?? "")}
              searchable
              w={300}
            />
            <Group gap="sm">
              {ownerDirty && (
                <Button
                  variant="subtle"
                  size="sm"
                  onClick={() =>
                    setSelectedOwner(ownerId != null ? String(ownerId) : "")
                  }
                >
                  Reset
                </Button>
              )}
              <Button
                size="sm"
                disabled={!ownerDirty || updateDevice.isPending}
                loading={updateDevice.isPending}
                onClick={handleOwnerSave}
              >
                Save
              </Button>
            </Group>
          </>
        )}
      </Stack>
    </Card>
  );
}

// ---------------------------------------------------------------------------
// Main tab
// ---------------------------------------------------------------------------

type LeaseRuleFormValues = { value: string; unit: TtlUnit };

const leaseRuleFormSchema = z.object({
  value: z.coerce.number().int("Must be a whole number").min(1, "Minimum is 1"),
  unit: z.enum(TTL_UNITS),
});

type MaxAddressesFormValues = { max_addresses: string };

const maxAddressesFormSchema = z.object({
  max_addresses: z.coerce
    .number()
    .int("Must be a whole number")
    .min(1, "Minimum is 1"),
});

interface DeviceSettingsTabProps {
  deviceId: number;
  device?: {
    name: string;
    api_key_prefix: string;
    device_type: DeviceType;
    description?: string | null;
    icon?: string | null;
    owner_id?: number;
    owner_name?: string;
  };
}

export function DeviceSettingsTab({
  deviceId,
  device,
}: DeviceSettingsTabProps) {
  const { data: deviceTypes } = useDeviceTypes();

  const {
    data: addressLeaseRule,
    isLoading: isAddressLeaseLoading,
    isError: isAddressLeaseError,
    error: addressLeaseError,
  } = useDeviceAddressLeaseRule(deviceId);
  const putRuleMutation = usePutDeviceAddressLeaseRule(deviceId);
  const disableRuleMutation = useDisableDeviceAddressLeaseRule(deviceId);
  const regenerateApiKey = useRegenerateApiKey();

  const {
    data: maxAddressesRule,
    isLoading: isMaxAddressesLoading,
    isError: isMaxAddressesError,
    error: maxAddressesError,
  } = useMaxActiveAddressesRule(deviceId);
  const putMaxAddressesRuleMutation = usePutMaxActiveAddressesRule(deviceId);
  const disableMaxAddressesRuleMutation =
    useDisableMaxActiveAddressesRule(deviceId);

  const [regeneratedApiKey, setRegeneratedApiKey] = useState<string | null>(
    null,
  );
  const [confirmRegenOpen, setConfirmRegenOpen] = useState(false);

  const leaseRuleForm = useForm<LeaseRuleFormValues>({
    validate: schemaResolver(leaseRuleFormSchema),
    initialValues: { value: "5", unit: "minutes" },
  });
  const { setValues: setAddressLeaseValues } = leaseRuleForm;
  const [addressLeaseEditing, setAddressLeaseEditing] = useState(false);

  const isAddressLeaseOn = Boolean(addressLeaseRule && addressLeaseRule.enabled);

  const maxAddressesForm = useForm<MaxAddressesFormValues>({
    validate: schemaResolver(maxAddressesFormSchema),
    initialValues: { max_addresses: "2" },
  });
  const { setValues: setMaxAddressesValues } = maxAddressesForm;
  const [maxAddressesEditing, setMaxAddressesEditing] = useState(false);

  const isMaxAddressesOn = Boolean(
    maxAddressesRule && maxAddressesRule.enabled,
  );

  async function handleCopyRegeneratedKey() {
    if (!regeneratedApiKey) return;
    if (!("clipboard" in navigator) || !navigator.clipboard?.writeText) {
      notifications.show({
        message: "Copy to clipboard is not supported in this browser.",
        color: "red",
      });
      return;
    }
    try {
      await navigator.clipboard.writeText(regeneratedApiKey);
      notifications.show({ message: "Copied to clipboard", color: "green" });
    } catch {
      notifications.show({ message: "Failed to copy API key", color: "red" });
    }
  }

  function handleConfirmRegenerate() {
    regenerateApiKey.mutate(
      { path: { device_id: deviceId } },
      {
        onSuccess: (data) => {
          setConfirmRegenOpen(false);
          setRegeneratedApiKey(data.api_key);
        },
        onError: (err) =>
          notifications.show({ color: "red", message: toErrorMessage(err) }),
      },
    );
  }

  function handleAddressLeaseSubmit(values: LeaseRuleFormValues) {
    putRuleMutation.mutate(
      {
        path: { device_id: deviceId },
        body: { ttl_seconds: toSeconds(Number(values.value), values.unit) },
      },
      {
        onSuccess: () => {
          setAddressLeaseEditing(false);
          notifications.show({
            color: "green",
            message: "Address lease rule saved",
          });
        },
        onError: (err) =>
          notifications.show({
            color: "red",
            title: "Error",
            message: toErrorMessage(err),
          }),
      },
    );
  }

  function handleAddressLeaseStartEditing() {
    if (!addressLeaseRule) return;
    setAddressLeaseValues(fromSeconds(addressLeaseRule.ttl_seconds));
    setAddressLeaseEditing(true);
  }

  useEffect(() => {
    if (!addressLeaseRule || isAddressLeaseOn) return;
    setAddressLeaseValues(fromSeconds(addressLeaseRule.ttl_seconds));
  }, [isAddressLeaseOn, addressLeaseRule, setAddressLeaseValues]);

  useEffect(() => {
    if (!maxAddressesRule || isMaxAddressesOn) return;
    setMaxAddressesValues({
      max_addresses: String(maxAddressesRule.max_addresses),
    });
  }, [isMaxAddressesOn, maxAddressesRule, setMaxAddressesValues]);

  function handleMaxAddressesSubmit(values: MaxAddressesFormValues) {
    putMaxAddressesRuleMutation.mutate(
      {
        path: { device_id: deviceId },
        body: { max_addresses: Number(values.max_addresses) },
      },
      {
        onSuccess: () => {
          setMaxAddressesEditing(false);
          notifications.show({
            color: "green",
            message: "Max active IPs rule saved",
          });
        },
        onError: (err) =>
          notifications.show({
            color: "red",
            title: "Error",
            message: toErrorMessage(err),
          }),
      },
    );
  }

  function handleStartEditingMaxAddresses() {
    if (!maxAddressesRule) return;
    setMaxAddressesValues({
      max_addresses: String(maxAddressesRule.max_addresses),
    });
    setMaxAddressesEditing(true);
  }

  const ttlLabel =
    addressLeaseRule && addressLeaseRule.ttl_seconds
      ? formatTtlLabel(addressLeaseRule.ttl_seconds)
      : null;
  const submitButtonLabel = putRuleMutation.isPending
    ? "Saving..."
    : isAddressLeaseOn
      ? "Save"
      : "Enable auto-expiry";

  const maxAddressesSubmitLabel = putMaxAddressesRuleMutation.isPending
    ? "Saving..."
    : isMaxAddressesOn
      ? "Save"
      : "Enable max-IP rule";

  return (
    <Stack gap="xl">
      {/* Device profile */}
      <Stack gap="sm">
        <Title order={5}>Device profile</Title>
        {device ? (
          <DeviceProfileCard
            deviceId={deviceId}
            device={device}
            deviceTypes={deviceTypes ?? []}
          />
        ) : (
          <Card withBorder>
            <Stack gap={8}>
              <Skeleton height={36} />
              <Skeleton height={36} />
              <Skeleton height={60} />
            </Stack>
          </Card>
        )}
      </Stack>

      {/* Ownership */}
      <Stack gap="sm">
        <Title order={5}>Ownership</Title>
        {device ? (
          <DeviceOwnershipCard
            key={device.owner_id}
            deviceId={deviceId}
            ownerId={device.owner_id}
            ownerName={device.owner_name}
          />
        ) : (
          <Card withBorder>
            <Skeleton height={20} width={180} />
          </Card>
        )}
      </Stack>

      {/* Settings */}
      <Stack gap="sm">
        <Title order={5}>Settings</Title>
        <Card withBorder>
          <Group justify="space-between" gap="md">
            <Stack gap={4}>
              <Text size="sm" fw={500}>
                API Key
              </Text>
              {device ? (
                <Text ff="monospace" size="sm" c="dimmed">
                  {device.api_key_prefix}&hellip;
                </Text>
              ) : (
                <Skeleton height={16} width={128} />
              )}
            </Stack>
            <Button
              variant="outline"
              size="sm"
              disabled={!device || regenerateApiKey.isPending}
              onClick={() => setConfirmRegenOpen(true)}
            >
              Regenerate API key
            </Button>
          </Group>
        </Card>
      </Stack>

      {/* Rules */}
      <Stack gap="sm">
        <Title order={5}>Rules</Title>
        <Card withBorder>
          <Title order={4} mb="md">
            Auto-expiry rule
          </Title>
          {isAddressLeaseLoading ? (
            <Stack gap={8}>
              <Skeleton height={16} width={160} />
              <Skeleton height={16} width={256} />
            </Stack>
          ) : isAddressLeaseError ? (
            <Text size="sm" c="red">
              Error loading rule: {toErrorMessage(addressLeaseError)}
            </Text>
          ) : (
            <Stack gap="md">
              {isAddressLeaseOn && (
                <Stack gap={4}>
                  <Group gap="sm">
                    <Text size="sm">Status:</Text>
                    <Badge color="green" variant="light" size="sm">
                      Enabled
                    </Badge>
                  </Group>
                  {ttlLabel && (
                    <Group gap="sm">
                      <Text size="sm">TTL:</Text>
                      <Text size="sm" fw={600}>
                        {ttlLabel}
                      </Text>
                    </Group>
                  )}
                </Stack>
              )}
              {!isAddressLeaseOn && (
                <Group gap="sm">
                  <Text size="sm">Status:</Text>
                  <Badge color="red" variant="light" size="sm">
                    Disabled
                  </Badge>
                  <Text size="sm" c="dimmed">
                    Turn it on to automatically revoke stale addresses.
                  </Text>
                </Group>
              )}
              {(!isAddressLeaseOn || addressLeaseEditing) && (
                <form
                  onSubmit={leaseRuleForm.onSubmit(handleAddressLeaseSubmit)}
                >
                  <Group align="flex-end" gap="md" wrap="wrap">
                    <TextInput
                      label="Expires after"
                      type="number"
                      min={1}
                      step={1}
                      placeholder="1"
                      w={128}
                      {...leaseRuleForm.getInputProps("value")}
                    />
                    <NativeSelect
                      label="Unit"
                      w={128}
                      data={TTL_UNITS.map((unit) => ({
                        label: unit,
                        value: unit,
                      }))}
                      {...leaseRuleForm.getInputProps("unit")}
                    />
                    <Button type="submit" disabled={putRuleMutation.isPending}>
                      {submitButtonLabel}
                    </Button>
                    {addressLeaseEditing && (
                      <Button
                        type="button"
                        variant="outline"
                        onClick={() => setAddressLeaseEditing(false)}
                      >
                        Cancel
                      </Button>
                    )}
                  </Group>
                </form>
              )}
              {isAddressLeaseOn && !addressLeaseEditing && (
                <Group gap="sm" wrap="wrap">
                  <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    onClick={handleAddressLeaseStartEditing}
                  >
                    Change TTL
                  </Button>
                  <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    onClick={() =>
                      disableRuleMutation.mutate(
                        { path: { device_id: deviceId } },
                        {
                          onSuccess: () =>
                            notifications.show({
                              color: "green",
                              message: "Address lease rule disabled",
                            }),
                          onError: (err) =>
                            notifications.show({
                              color: "red",
                              title: "Error",
                              message: toErrorMessage(err),
                            }),
                        },
                      )
                    }
                    disabled={disableRuleMutation.isPending}
                  >
                    Turn off auto-expiry
                  </Button>
                </Group>
              )}
            </Stack>
          )}
        </Card>
        <Card withBorder>
          <Title order={4} mb="md">
            Max active IPs rule
          </Title>
          {isMaxAddressesLoading ? (
            <Stack gap={8}>
              <Skeleton height={16} width={160} />
              <Skeleton height={16} width={256} />
            </Stack>
          ) : isMaxAddressesError ? (
            <Text size="sm" c="red">
              Error loading rule: {toErrorMessage(maxAddressesError)}
            </Text>
          ) : (
            <Stack gap="md">
              {isMaxAddressesOn && (
                <Stack gap={4}>
                  <Group gap="sm">
                    <Text size="sm">Status:</Text>
                    <Badge color="green" variant="light" size="sm">
                      Enabled
                    </Badge>
                  </Group>
                  <Group gap="sm">
                    <Text size="sm">Max IPs:</Text>
                    <Text size="sm" fw={600}>
                      {maxAddressesRule!.max_addresses}
                    </Text>
                  </Group>
                </Stack>
              )}
              {!isMaxAddressesOn && (
                <Group gap="sm">
                  <Text size="sm">Status:</Text>
                  <Badge color="red" variant="light" size="sm">
                    Disabled
                  </Badge>
                  <Text size="sm" c="dimmed">
                    Turn it on to limit the number of simultaneously active IPs.
                  </Text>
                </Group>
              )}
              {(!isMaxAddressesOn || maxAddressesEditing) && (
                <form
                  onSubmit={maxAddressesForm.onSubmit(handleMaxAddressesSubmit)}
                >
                  <Group align="flex-end" gap="md" wrap="wrap">
                    <TextInput
                      label="Max active IPs"
                      type="number"
                      min={1}
                      step={1}
                      placeholder="3"
                      w={128}
                      {...maxAddressesForm.getInputProps("max_addresses")}
                    />
                    <Button
                      type="submit"
                      disabled={putMaxAddressesRuleMutation.isPending}
                    >
                      {maxAddressesSubmitLabel}
                    </Button>
                    {maxAddressesEditing && (
                      <Button
                        type="button"
                        variant="outline"
                        onClick={() => setMaxAddressesEditing(false)}
                      >
                        Cancel
                      </Button>
                    )}
                  </Group>
                </form>
              )}
              {isMaxAddressesOn && !maxAddressesEditing && (
                <Group gap="sm" wrap="wrap">
                  <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    onClick={handleStartEditingMaxAddresses}
                  >
                    Change limit
                  </Button>
                  <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    onClick={() =>
                      disableMaxAddressesRuleMutation.mutate(
                        { path: { device_id: deviceId } },
                        {
                          onSuccess: () =>
                            notifications.show({
                              color: "green",
                              message: "Max active IPs rule disabled",
                            }),
                          onError: (err) =>
                            notifications.show({
                              color: "red",
                              title: "Error",
                              message: toErrorMessage(err),
                            }),
                        },
                      )
                    }
                    disabled={disableMaxAddressesRuleMutation.isPending}
                  >
                    Turn off max-IP rule
                  </Button>
                </Group>
              )}
            </Stack>
          )}
        </Card>
      </Stack>

      {/* Confirm regenerate API key modal */}
      <Modal
        opened={confirmRegenOpen}
        onClose={() => setConfirmRegenOpen(false)}
        title={`Regenerate API key for "${device?.name}"?`}
        closeOnClickOutside={false}
        closeOnEscape={false}
        withCloseButton={false}
      >
        <Text size="sm">
          The current key (
          <Text component="span" ff="monospace">
            {device?.api_key_prefix}&hellip;
          </Text>
          ) will stop working immediately. You will need to update any scripts
          or services using this device.
        </Text>
        <Group justify="flex-end" mt="md" gap="sm">
          <Button variant="outline" onClick={() => setConfirmRegenOpen(false)}>
            Cancel
          </Button>
          <Button
            onClick={handleConfirmRegenerate}
            disabled={regenerateApiKey.isPending}
          >
            Regenerate
          </Button>
        </Group>
      </Modal>

      {/* One-time key display modal after successful regeneration */}
      <Modal
        opened={regeneratedApiKey !== null}
        onClose={() => setRegeneratedApiKey(null)}
        title="API key regenerated — save your new key"
        closeOnClickOutside={false}
        closeOnEscape={false}
        withCloseButton={false}
      >
        <Stack gap="md">
          <Text size="sm" c="dimmed">
            This API key is shown only once. Copy it now and store it securely.
            The old key is no longer valid.
          </Text>
          {regeneratedApiKey && (
            <>
              <Stack gap={8}>
                <Text size="sm" fw={500}>
                  New API key
                </Text>
                <Group gap="sm">
                  <TextInput
                    readOnly
                    value={regeneratedApiKey}
                    ff="monospace"
                    style={{ flex: 1 }}
                  />
                  <Button
                    type="button"
                    variant="outline"
                    onClick={handleCopyRegeneratedKey}
                  >
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
            <Button type="button" onClick={() => setRegeneratedApiKey(null)}>
              I&apos;ve saved it
            </Button>
          </Group>
        </Stack>
      </Modal>
    </Stack>
  );
}
