import { useState } from "react";
import { Button, Group, Modal, Stack, TextInput, Textarea } from "@mantine/core";
import type { MantineColor } from "@mantine/core";
import { IconPicker } from "@/features/host-access/components/IconPicker";
import { GroupColorPicker } from "@/features/host-access/components/GroupColorPicker";
import type { DraftGroup, GroupColor } from "@/features/host-access/drafts/hostGroupsDraft";
export type GroupFormValues = {
  name: string;
  description: string | null;
  icon: string | null;
  color: GroupColor | null;
};

interface Props {
  opened: boolean;
  onClose: () => void;
  initial?: DraftGroup | null;
  existingNames: string[];
  onSubmit: (values: GroupFormValues) => void;
}

export function GroupMetadataModal({
  opened,
  onClose,
  initial,
  existingNames,
  onSubmit,
}: Props) {
  const isEditing = initial !== null && initial !== undefined;
  return (
    <Modal
      opened={opened}
      onClose={onClose}
      title={isEditing ? `Edit group — ${initial?.name}` : "New host group"}
      size="lg"
    >
      {opened && (
        <GroupMetadataForm
          initial={initial ?? null}
          existingNames={existingNames}
          onSubmit={(values) => {
            onSubmit(values);
            onClose();
          }}
          onCancel={onClose}
        />
      )}
    </Modal>
  );
}

interface FormProps {
  initial: DraftGroup | null;
  existingNames: string[];
  onSubmit: (values: GroupFormValues) => void;
  onCancel: () => void;
}

function GroupMetadataForm({ initial, existingNames, onSubmit, onCancel }: FormProps) {
  const isEditing = initial !== null;
  const [name, setName] = useState(initial?.name ?? "");
  const [description, setDescription] = useState(initial?.description ?? "");
  const [icon, setIcon] = useState<string | null>(initial?.icon ?? null);
  const [color, setColor] = useState<GroupColor | null>(initial?.color ?? null);

  const trimmed = name.trim();
  const nameTaken =
    trimmed.length > 0 &&
    existingNames.some(
      (n) => n.toLowerCase() === trimmed.toLowerCase() && n !== initial?.name,
    );
  const canSubmit = trimmed.length > 0 && !nameTaken;
  const previewColor: MantineColor = color ?? "yellow";

  function handleSubmit() {
    if (!canSubmit) return;
    onSubmit({
      name: trimmed,
      description: description.trim() || null,
      icon,
      color,
    });
  }

  return (
    <Stack gap="md">
      <TextInput
        label="Name"
        placeholder="e.g. Media"
        value={name}
        onChange={(e) => setName(e.currentTarget.value)}
        required
        autoFocus={!isEditing}
        error={nameTaken ? "A group with this name already exists" : null}
      />
      <Textarea
        label="Description"
        placeholder="Optional"
        value={description}
        onChange={(e) => setDescription(e.currentTarget.value)}
        rows={2}
      />
      <GroupColorPicker value={color} onChange={setColor} />
      <IconPicker value={icon} onChange={setIcon} color={previewColor} />

      <Group justify="flex-end" gap="xs">
        <Button variant="outline" onClick={onCancel}>
          Cancel
        </Button>
        <Button onClick={handleSubmit} disabled={!canSubmit}>
          {isEditing ? "Apply" : "Create"}
        </Button>
      </Group>
    </Stack>
  );
}
