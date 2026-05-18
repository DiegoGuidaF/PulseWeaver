import { useState } from "react";
import { Button, Group, Modal, MultiSelect, Stack, Text, TextInput } from "@mantine/core";

export interface AddHostValues {
  fqdn: string;
  groupIds: number[];
}

interface Props {
  opened: boolean;
  onClose: () => void;
  groups: { id: number; name: string }[];
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
  groups: { id: number; name: string }[];
  existingFqdns: string[];
  onSubmit: (values: AddHostValues) => void;
  onCancel: () => void;
}

function AddHostForm({ groups, existingFqdns, onSubmit, onCancel }: FormProps) {
  const [fqdn, setFqdn] = useState("");
  const [groupIds, setGroupIds] = useState<string[]>([]);

  const trimmed = fqdn.trim().toLowerCase();
  const duplicate = trimmed.length > 0 && existingFqdns.some((f) => f.toLowerCase() === trimmed);
  const canSubmit = trimmed.length > 0 && !duplicate;

  function handleSubmit() {
    if (!canSubmit) return;
    onSubmit({ fqdn: trimmed, groupIds: groupIds.map(Number) });
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
        <MultiSelect
          label="Groups"
          description="Optional — pick any existing groups this host should join."
          placeholder="Search groups…"
          data={groups.map((g) => ({ value: String(g.id), label: g.name }))}
          value={groupIds}
          onChange={setGroupIds}
          searchable
          clearable
        />
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
