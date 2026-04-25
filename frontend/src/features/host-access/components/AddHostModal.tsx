import { useState } from "react";
import { Button, Group, Modal, MultiSelect, Stack, TextInput } from "@mantine/core";
import { IconPicker } from "@/features/host-access/components/IconPicker";
import type { DraftGroup } from "@/features/host-access/drafts/hostGroupsDraft";
import type { Id } from "@/lib/api";

export interface AddHostValues {
  fqdn: string;
  icon: string | null;
  groupIds: Id[];
}

interface Props {
  opened: boolean;
  onClose: () => void;
  groups: DraftGroup[];
  existingFqdns: string[];
  onSubmit: (values: AddHostValues) => void;
}

export function AddHostModal({ opened, onClose, groups, existingFqdns, onSubmit }: Props) {
  return (
    <Modal opened={opened} onClose={onClose} title="New known host" size="lg">
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
  groups: DraftGroup[];
  existingFqdns: string[];
  onSubmit: (values: AddHostValues) => void;
  onCancel: () => void;
}

function AddHostForm({ groups, existingFqdns, onSubmit, onCancel }: FormProps) {
  const [fqdn, setFqdn] = useState("");
  const [icon, setIcon] = useState<string | null>(null);
  const [groupIds, setGroupIds] = useState<string[]>([]);

  const trimmed = fqdn.trim().toLowerCase();
  const duplicate =
    trimmed.length > 0 && existingFqdns.some((f) => f.toLowerCase() === trimmed);
  const canSubmit = trimmed.length > 0 && !duplicate;

  const groupOptions = groups
    .filter((g) => typeof g.id === "number")
    .map((g) => ({ value: String(g.id), label: g.name || "Unnamed group" }));

  function handleSubmit() {
    if (!canSubmit) return;
    onSubmit({
      fqdn: trimmed,
      icon,
      groupIds: groupIds.map((s) => Number(s)),
    });
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
        onKeyDown={(e) => {
          if (e.key === "Enter") handleSubmit();
        }}
      />
      <MultiSelect
        label="Groups"
        description="Optional — assign to one or more host groups."
        placeholder="Pick groups"
        data={groupOptions}
        value={groupIds}
        onChange={setGroupIds}
        searchable
        clearable
      />
      <IconPicker value={icon} onChange={setIcon} />

      <Group justify="flex-end" gap="xs">
        <Button variant="outline" onClick={onCancel}>
          Cancel
        </Button>
        <Button onClick={handleSubmit} disabled={!canSubmit}>
          Stage host
        </Button>
      </Group>
    </Stack>
  );
}
