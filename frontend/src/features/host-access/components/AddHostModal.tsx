import { useState } from "react";
import { Button, Group, Modal, Stack, Text, TextInput } from "@mantine/core";
import { GroupChipPicker } from "@/features/host-access/components/GroupChipPicker";

export interface AddHostValues {
  fqdn: string;
  groupIds: number[];
}

interface PickableGroup {
  id: number;
  name: string;
  color?: string | null;
  icon?: string | null;
}

interface Props {
  opened: boolean;
  onClose: () => void;
  groups: PickableGroup[];
  existingFqdns: string[];
  onSubmit: (values: AddHostValues) => void;
}

export function AddHostModal({ opened, onClose, groups, existingFqdns, onSubmit }: Props) {
  return (
    <Modal opened={opened} onClose={onClose} title="New host" size="md">
      {opened && (
        <AddHostForm
          groups={groups}
          existingFqdns={existingFqdns}
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
  groups: PickableGroup[];
  existingFqdns: string[];
  onSubmit: (values: AddHostValues) => void;
  onCancel: () => void;
}

function AddHostForm({ groups, existingFqdns, onSubmit, onCancel }: FormProps) {
  const [fqdn, setFqdn] = useState("");
  const [groupIds, setGroupIds] = useState<Set<number>>(new Set());

  const trimmed = fqdn.trim().toLowerCase();
  const duplicate = trimmed.length > 0 && existingFqdns.some((f) => f.toLowerCase() === trimmed);
  const canSubmit = trimmed.length > 0 && !duplicate;

  function handleSubmit() {
    if (!canSubmit) return;
    onSubmit({ fqdn: trimmed, groupIds: [...groupIds] });
  }

  return (
    <Stack gap="md">
      <TextInput
        label="FQDN"
        description="Exact match — no wildcards."
        placeholder="e.g. jellyfin.myhome.org"
        value={fqdn}
        onChange={(e) => setFqdn(e.currentTarget.value)}
        ff="monospace"
        autoFocus
        error={duplicate ? "This host is already in the list" : null}
        onKeyDown={(e) => { if (e.key === "Enter") handleSubmit(); }}
      />

      {groups.length > 0 && (
        <Stack gap={4}>
          <Text size="sm" fw={500}>Groups</Text>
          <Text size="xs" c="dimmed" mt={-4}>
            Optional — pick any existing groups this host should join.
          </Text>
          <Group gap="xs" wrap="wrap">
            <GroupChipPicker
              availableGroups={groups}
              selected={groupIds}
              onChange={setGroupIds}
              emptyLabel="Add group"
              addLabel="+ group"
              removeAriaLabel={(name) => `Remove ${name}`}
            />
          </Group>
        </Stack>
      )}

      <Group justify="flex-end" gap="xs">
        <Button variant="outline" onClick={onCancel}>
          Cancel
        </Button>
        <Button onClick={handleSubmit} disabled={!canSubmit}>
          Add to draft
        </Button>
      </Group>
      <Text size="xs" c="dimmed" mt={-8}>
        Staged — click Save in the changes bar to commit.
      </Text>
    </Stack>
  );
}
