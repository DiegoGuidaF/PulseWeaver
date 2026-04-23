import { useState } from "react";
import { Badge, Button, Group, Modal, Stack, Text, Textarea } from "@mantine/core";
import { notifications } from "@mantine/notifications";
import { useCreateKnownHosts } from "@/features/host-access/hooks/useCreateKnownHosts";
import { toErrorMessage } from "@/lib/api-client";

interface Props {
  opened: boolean;
  onClose: () => void;
  existingFqdns: string[];
}

export function BulkImportModal({ opened, onClose, existingFqdns }: Props) {
  const [text, setText] = useState("");
  const createKnownHosts = useCreateKnownHosts();

  const parsed = text
    .split("\n")
    .map((s) => s.trim().toLowerCase())
    .filter(Boolean);
  const existing = new Set(existingFqdns);
  const toAdd = parsed.filter((s) => !existing.has(s));
  const duplicates = parsed.length - toAdd.length;

  function handleImport() {
    if (toAdd.length === 0) return;
    createKnownHosts.mutate(
      { body: { fqdns: toAdd } },
      {
        onSuccess: () => {
          notifications.show({
            color: "green",
            message: `${toAdd.length} host${toAdd.length !== 1 ? "s" : ""} imported${duplicates > 0 ? ` (${duplicates} duplicate${duplicates !== 1 ? "s" : ""} skipped)` : ""}`,
          });
          setText("");
          onClose();
        },
        onError: (err) =>
          notifications.show({ color: "red", title: "Import failed", message: toErrorMessage(err) }),
      },
    );
  }

  function handleClose() {
    setText("");
    onClose();
  }

  return (
    <Modal opened={opened} onClose={handleClose} title="Bulk import hosts" size="lg">
      <Stack gap="md">
        <Text size="sm" c="dimmed">
          One FQDN per line. Duplicates and blank lines are skipped.
        </Text>
        <Textarea
          rows={10}
          value={text}
          onChange={(e) => setText(e.currentTarget.value)}
          placeholder={"jellyfin.myhome.org\nnextcloud.myhome.org\nimmich.myhome.org"}
          ff="monospace"
          styles={{ input: { fontSize: 13 } }}
        />
        {parsed.length > 0 && (
          <Group gap="xs">
            <Badge color="indigo">{toAdd.length} to add</Badge>
            {duplicates > 0 && <Badge color="gray">{duplicates} duplicate{duplicates !== 1 ? "s" : ""}</Badge>}
          </Group>
        )}
        <Group justify="flex-end" gap="xs">
          <Button variant="outline" onClick={handleClose}>
            Cancel
          </Button>
          <Button
            onClick={handleImport}
            disabled={toAdd.length === 0 || createKnownHosts.isPending}
            loading={createKnownHosts.isPending}
          >
            Import {toAdd.length > 0 ? toAdd.length : ""}
          </Button>
        </Group>
      </Stack>
    </Modal>
  );
}
