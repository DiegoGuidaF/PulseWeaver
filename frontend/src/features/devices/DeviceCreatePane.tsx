import { useState } from "react";
import { schemaResolver, useForm } from "@mantine/form";
import { z } from "zod";
import {
  ActionIcon,
  Alert,
  Badge,
  Button,
  Card,
  Group,
  Radio,
  Stack,
  Text,
  Textarea,
  TextInput,
  ThemeIcon,
  Tooltip,
  UnstyledButton,
} from "@mantine/core";
import { notifications } from "@mantine/notifications";
import { IconInfoCircle, IconLock, IconX } from "@tabler/icons-react";
import { useCreateDevice } from "@/features/devices/hooks/useCreateDevice";
import { IconPickerPopover } from "@/features/devices/IconPickerPopover";
import { resolveDeviceIcon, suggestIcon } from "@/features/devices/deviceTypeConfig";
import { useClipboard } from "@/hooks/useClipboard";
import { toApiError, toErrorMessage } from "@/lib/api-client";
import { zCreateDeviceRequest } from "@/lib/api/zod.gen";

type Credential = "none" | "apikey" | "pairing";

const formSchema = z.object({
  name: zCreateDeviceRequest.shape.name,
  description: z.string().max(200),
  icon: z.string().max(80),
});

type FormValues = z.infer<typeof formSchema>;

const CREDENTIAL_OPTIONS: {
  value: Credential;
  label: string;
  description: string;
  tooltip: string;
}[] = [
  {
    value: "none",
    label: "None",
    description:
      "Create now, provision later. Addresses are managed manually — best for a stable-IP device.",
    tooltip:
      "A device with no credential is perfectly valid. You can add an API key or pairing code anytime later.",
  },
  {
    value: "apikey",
    label: "API key",
    description:
      "Mints a key now (shown once). The device updates its own addresses via the API — typical for a changing-IP device.",
    tooltip:
      "The key is shown a single time on creation. Store it securely; you can regenerate it later if lost.",
  },
  {
    value: "pairing",
    label: "Pairing code",
    description:
      "A one-time code the companion app claims. No key exists until the device pairs.",
    tooltip:
      "The API key is created on claim, not now — until the device pairs there is no key to leak.",
  },
];

interface DeviceCreatePaneProps {
  ownerId: number;
  ownerName: string;
  onCancel: () => void;
  /** Called once the device exists; `tab` is where the populated detail should land. */
  onCreated: (deviceId: number, tab: "addresses" | "pairing") => void;
}

export function DeviceCreatePane({
  ownerId,
  ownerName,
  onCancel,
  onCreated,
}: DeviceCreatePaneProps) {
  const createDevice = useCreateDevice();
  const { copy } = useClipboard();

  const [credential, setCredential] = useState<Credential>("none");
  const [iconPickerOpen, setIconPickerOpen] = useState(false);
  const [iconAutoSuggested, setIconAutoSuggested] = useState(true);
  // After an API-key device is created, hold the shown-once key in this pane.
  const [mintedKey, setMintedKey] = useState<{ deviceId: number; key: string } | null>(null);

  const form = useForm<FormValues>({
    validateInputOnBlur: true,
    validate: schemaResolver(formSchema),
    initialValues: { name: "", description: "", icon: "" },
  });

  const nameValue = form.values.name;
  // Icon is derived, not synced via an effect: until the user picks one, it
  // tracks the name suggestion; a manual pick pins `form.values.icon`.
  const effectiveIcon = iconAutoSuggested ? suggestIcon(nameValue) : form.values.icon;
  const renderIcon = resolveDeviceIcon(effectiveIcon || null);
  const isIconAutoSuggested = iconAutoSuggested && !!effectiveIcon;
  const rulesNudged = credential !== "none";

  function onSubmit(values: FormValues) {
    const icon = iconAutoSuggested ? suggestIcon(values.name) : values.icon;
    createDevice.mutate(
      {
        body: {
          name: values.name,
          owner_id: ownerId,
          description: values.description || null,
          icon: icon || null,
          generate_api_key: credential === "apikey",
        },
      },
      {
        onSuccess: (data) => {
          const deviceId = data.device.id;
          if (credential === "apikey" && data.api_key) {
            setMintedKey({ deviceId, key: data.api_key });
            return;
          }
          onCreated(deviceId, credential === "pairing" ? "pairing" : "addresses");
        },
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

  if (mintedKey) {
    return (
      <Stack gap="lg" maw={640}>
        <div>
          <Text fw={600} size="lg">{form.values.name} created</Text>
          <Text size="sm" c="dimmed">Save the API key now — it is shown only once.</Text>
        </div>
        <Card withBorder>
          <Stack gap={8}>
            <Text size="sm" fw={500}>API key</Text>
            <Group gap="sm">
              <TextInput readOnly value={mintedKey.key} ff="monospace" style={{ flex: 1 }} />
              <Button
                variant="outline"
                onClick={() => copy(mintedKey.key, { errorMessage: "Failed to copy API key" })}
              >
                Copy
              </Button>
            </Group>
            <Text size="xs" c="dimmed">
              You will not be able to see this key again. Address rules are recommended for an
              API-key device — set them on the device&apos;s Rules tab.
            </Text>
          </Stack>
        </Card>
        <Group justify="flex-end">
          <Button onClick={() => onCreated(mintedKey.deviceId, "addresses")}>
            I&apos;ve saved it — open device
          </Button>
        </Group>
      </Stack>
    );
  }

  return (
    <form onSubmit={form.onSubmit(onSubmit)}>
      <Stack gap="xl" maw={640}>
        <Group justify="space-between" align="center">
          <Text fw={600} size="lg">New device for {ownerName}</Text>
          <Tooltip label="Pick the user first, then create — owner can't be changed here" withArrow>
            <Badge
              variant="light"
              color="gray"
              leftSection={<IconLock size={11} />}
              style={{ cursor: "help" }}
            >
              owner: {ownerName}
            </Badge>
          </Tooltip>
        </Group>

        {/* ── Device ── */}
        <Stack gap="sm">
          <Text size="xs" fw={700} tt="uppercase" c="dimmed" style={{ letterSpacing: "0.05em" }}>
            Device
          </Text>
          <Group align="flex-start" gap="md" wrap="nowrap">
            <TextInput
              label="Name"
              placeholder="e.g. Office Printer"
              data-autofocus
              withAsterisk
              style={{ flex: 1 }}
              {...form.getInputProps("name")}
            />
            <div>
              <Text size="sm" fw={500} mb={4}>Icon</Text>
              <Group gap="xs" align="center">
                <IconPickerPopover
                  opened={iconPickerOpen}
                  onClose={() => setIconPickerOpen(false)}
                  selectedIcon={effectiveIcon}
                  onSelect={(name) => {
                    setIconAutoSuggested(false);
                    form.setFieldValue("icon", name);
                  }}
                  deviceName={nameValue}
                  target={
                    <Tooltip label={effectiveIcon || "default"} withArrow>
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
                {!iconAutoSuggested && (
                  <ActionIcon
                    variant="subtle"
                    color="dimmed"
                    size="sm"
                    aria-label="Clear icon override"
                    onClick={() => {
                      setIconAutoSuggested(true);
                      form.setFieldValue("icon", "");
                    }}
                  >
                    <IconX size={14} />
                  </ActionIcon>
                )}
              </Group>
              {isIconAutoSuggested && (
                <Text size="xs" c="dimmed" mt={4}>from name</Text>
              )}
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
              {form.values.description.length}/200
            </Text>
          </div>
        </Stack>

        {/* ── Credential ── */}
        <Stack gap="sm">
          <Text size="xs" fw={700} tt="uppercase" c="dimmed" style={{ letterSpacing: "0.05em" }}>
            Credential — how this device gets its access key
          </Text>
          <Radio.Group
            value={credential}
            onChange={(val) => setCredential(val as Credential)}
          >
            <Stack gap="sm">
              {CREDENTIAL_OPTIONS.map((opt) => (
                <Radio
                  key={opt.value}
                  value={opt.value}
                  description={opt.description}
                  label={
                    <Group gap={4} align="center">
                      <span>{opt.label}</span>
                      <Tooltip label={opt.tooltip} multiline w={280} withArrow>
                        <IconInfoCircle
                          size={13}
                          style={{ color: "var(--mantine-color-dimmed)", cursor: "help" }}
                        />
                      </Tooltip>
                    </Group>
                  }
                />
              ))}
            </Stack>
          </Radio.Group>
        </Stack>

        {/* ── Rules nudge ── */}
        {rulesNudged ? (
          <Alert color="orange" variant="light" icon={<IconInfoCircle size={16} />}>
            <Text size="sm">
              This device updates its own addresses, so its live IPs can pile up. Set
              <strong> address rules</strong> (TTL + max live addresses) on the device&apos;s Rules
              tab after creating — recommended for a changing-IP device.
            </Text>
          </Alert>
        ) : (
          <Text size="xs" c="dimmed">
            Addresses are managed manually for a credential-less device. Address rules are optional
            and available on the device later.
          </Text>
        )}

        <Group justify="flex-end" gap="sm">
          <Button type="button" variant="outline" onClick={onCancel}>
            Cancel
          </Button>
          <Button type="submit" loading={createDevice.isPending}>
            Create device
          </Button>
        </Group>
      </Stack>
    </form>
  );
}

/** Empty-state entry point shown in the right pane when an owner has no devices. */
export function DeviceCreateEmptyState({
  ownerName,
  onCreate,
}: {
  ownerName: string;
  onCreate: () => void;
}) {
  return (
    <Card withBorder maw={560}>
      <Stack gap="sm" align="flex-start">
        <ThemeIcon variant="light" color="gray" size="lg">
          {resolveDeviceIcon(null)({ size: 20 })}
        </ThemeIcon>
        <Text fw={600}>No devices for {ownerName} yet</Text>
        <Text size="sm" c="dimmed">
          Create the device now; provision a credential (API key or pairing code) now or later —
          a device with no credential is perfectly valid.
        </Text>
        <Button onClick={onCreate}>+ Create device</Button>
      </Stack>
    </Card>
  );
}
