import { useState } from "react";
import {
  Button,
  Checkbox,
  Group,
  Modal,
  ScrollArea,
  Stack,
  Text,
  TextInput,
  Textarea,
} from "@mantine/core";
import { notifications } from "@mantine/notifications";
import type { HostGroupWithMembers, KnownHostWithStats } from "@/lib/api";
import { useCreateHostGroup } from "@/features/host-access/hooks/useCreateHostGroup";
import { useUpdateHostGroup } from "@/features/host-access/hooks/useUpdateHostGroup";
import { toErrorMessage } from "@/lib/api-client";

interface Props {
  opened: boolean;
  onClose: () => void;
  hosts: KnownHostWithStats[];
  editingGroup?: HostGroupWithMembers;
}

export function GroupFormModal({ opened, onClose, hosts, editingGroup }: Props) {
  const isEditing = editingGroup !== undefined;

  const [name, setName] = useState(editingGroup?.name ?? "");
  const [description, setDescription] = useState(editingGroup?.description ?? "");
  const [icon, setIcon] = useState(editingGroup?.icon ?? "");
  const [selectedHostIds, setSelectedHostIds] = useState<Set<number>>(
    () => new Set(editingGroup?.hosts.map((h) => h.id) ?? []),
  );

  const createHostGroup = useCreateHostGroup();
  const updateHostGroup = useUpdateHostGroup();
  const isPending = createHostGroup.isPending || updateHostGroup.isPending;

  function toggleHost(id: number) {
    setSelectedHostIds((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  }

  function handleSubmit() {
    const trimmedName = name.trim();
    if (!trimmedName) return;

    const body = {
      name: trimmedName,
      description: description.trim() || null,
      icon: icon.trim() || null,
      host_ids: [...selectedHostIds],
    };

    if (isEditing) {
      updateHostGroup.mutate(
        { path: { group_id: editingGroup.id }, body },
        {
          onSuccess: () => {
            notifications.show({ color: "green", message: `Group "${trimmedName}" updated` });
            onClose();
          },
          onError: (err) =>
            notifications.show({ color: "red", title: "Failed to update group", message: toErrorMessage(err) }),
        },
      );
    } else {
      createHostGroup.mutate(
        { body },
        {
          onSuccess: () => {
            notifications.show({ color: "green", message: `Group "${trimmedName}" created` });
            onClose();
          },
          onError: (err) =>
            notifications.show({ color: "red", title: "Failed to create group", message: toErrorMessage(err) }),
        },
      );
    }
  }

  return (
    <Modal
      opened={opened}
      onClose={onClose}
      title={isEditing ? `Edit group — ${editingGroup?.name}` : "New host group"}
      size="lg"
    >
      <Stack gap="md">
        <TextInput
          label="Name"
          placeholder="e.g. Media"
          value={name}
          onChange={(e) => setName(e.currentTarget.value)}
          required
          autoFocus={!isEditing}
        />
        <Textarea
          label="Description"
          placeholder="Optional description"
          value={description}
          onChange={(e) => setDescription(e.currentTarget.value)}
          rows={2}
        />
        <TextInput
          label="Icon"
          description="Tabler icon name (e.g. folder, server, lock). Leave empty for default."
          placeholder="e.g. folder"
          value={icon}
          onChange={(e) => setIcon(e.currentTarget.value)}
        />

        <div>
          <Text size="sm" fw={500} mb={4}>
            Hosts in group
          </Text>
          <Text size="xs" c="dimmed" mb={8}>
            Pick any subset of known hosts.
          </Text>
          {hosts.length === 0 ? (
            <Text size="sm" c="dimmed">
              No known hosts yet.
            </Text>
          ) : (
            <ScrollArea.Autosize mah={240}>
              <Stack gap={0} style={{ border: "1px solid var(--mantine-color-default-border)", borderRadius: 6 }}>
                {hosts.map((h, i) => (
                  <label
                    key={h.id}
                    style={{
                      display: "flex",
                      alignItems: "center",
                      gap: 10,
                      padding: "8px 12px",
                      borderBottom:
                        i < hosts.length - 1 ? "1px solid var(--mantine-color-default-border)" : "none",
                      cursor: "pointer",
                    }}
                  >
                    <Checkbox
                      checked={selectedHostIds.has(h.id)}
                      onChange={() => toggleHost(h.id)}
                      styles={{ input: { cursor: "pointer" } }}
                    />
                    <Text size="sm" ff="monospace">
                      {h.fqdn}
                    </Text>
                  </label>
                ))}
              </Stack>
            </ScrollArea.Autosize>
          )}
        </div>

        <Group justify="flex-end" gap="xs">
          <Button variant="outline" onClick={onClose}>
            Cancel
          </Button>
          <Button onClick={handleSubmit} disabled={!name.trim() || isPending} loading={isPending}>
            {isEditing ? "Save changes" : "Create group"}
          </Button>
        </Group>
      </Stack>
    </Modal>
  );
}
