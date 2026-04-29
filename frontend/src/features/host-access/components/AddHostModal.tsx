import { useState } from "react";
import { Box, Button, Chip, Group, Modal, Stack, Text, TextInput } from "@mantine/core";
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
    <Modal opened={opened} onClose={onClose} title="New known host" size="md">
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

  const assignableGroups = groups.filter((g) => typeof g.id === "number");

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

      <IconPicker value={icon} onChange={setIcon} color="gray" />

      {assignableGroups.length > 0 && (
        <Stack gap={6}>
          <Box>
            <Text size="sm" fw={500}>
              Add to groups
            </Text>
            <Text size="xs" c="dimmed">
              Optional — pick any existing groups this host should join.
            </Text>
          </Box>
          <Chip.Group multiple value={groupIds} onChange={setGroupIds}>
            <Group gap="xs">
              {assignableGroups.map((g) => (
                <Chip
                  key={g.id}
                  value={String(g.id)}
                  color={g.color ?? "yellow"}
                  size="sm"
                >
                  {g.name || "Unnamed group"}
                </Chip>
              ))}
            </Group>
          </Chip.Group>
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
