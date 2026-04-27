import { useState } from "react";
import {
  Alert,
  Badge,
  Button,
  Card,
  Checkbox,
  Group,
  Stack,
  Table,
  Text,
  Title,
} from "@mantine/core";
import { notifications } from "@mantine/notifications";
import { IconAlertCircle } from "@tabler/icons-react";
import type { HostSuggestionsPage } from "@/lib/api";
import { useIgnoreSuggestion } from "@/features/host-access/hooks/useIgnoreSuggestion";
import { useUnignoreSuggestion } from "@/features/host-access/hooks/useUnignoreSuggestion";
import { useDateFormatter } from "@/contexts/useDateTimePrefs";
import { toErrorMessage } from "@/lib/api-client";

interface Props {
  data: HostSuggestionsPage;
  locked: boolean;
  onDiscardLock: () => void;
  onStageHosts: (fqdns: string[]) => void;
}

export function SuggestionsTab({ data, locked, onDiscardLock, onStageHosts }: Props) {
  const formatDateTime = useDateFormatter();
  const ignoreSuggestion = useIgnoreSuggestion();
  const unignoreSuggestion = useUnignoreSuggestion();

  const [selected, setSelected] = useState<Set<string>>(new Set());

  const allFqdns = data.suggestions.map((s) => s.fqdn);
  const allSelected = allFqdns.length > 0 && allFqdns.every((f) => selected.has(f));
  const someSelected = selected.size > 0;

  function toggleAll() {
    if (allSelected) {
      setSelected(new Set());
    } else {
      setSelected(new Set(allFqdns));
    }
  }

  function toggleOne(fqdn: string) {
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(fqdn)) next.delete(fqdn);
      else next.add(fqdn);
      return next;
    });
  }

  function handlePromote(fqdn: string) {
    onStageHosts([fqdn]);
  }

  function handleIgnore(fqdn: string) {
    ignoreSuggestion.mutate(
      { body: { fqdn } },
      {
        onSuccess: () =>
          notifications.show({ color: "gray", message: `${fqdn} ignored` }),
        onError: (err: unknown) =>
          notifications.show({ color: "red", title: "Failed to ignore", message: toErrorMessage(err) }),
      },
    );
  }

  function handleUnignore(fqdn: string) {
    unignoreSuggestion.mutate(
      { path: { fqdn } },
      {
        onSuccess: () =>
          notifications.show({ color: "green", message: `${fqdn} removed from ignore list` }),
        onError: (err: unknown) =>
          notifications.show({ color: "red", title: "Failed to unignore", message: toErrorMessage(err) }),
      },
    );
  }

  function handleBulkAccept() {
    const fqdns = [...selected];
    onStageHosts(fqdns);
    setSelected(new Set());
  }

  async function handleBulkIgnore() {
    const fqdns = [...selected];
    let failed = 0;
    for (const fqdn of fqdns) {
      await new Promise<void>((resolve) => {
        ignoreSuggestion.mutate(
          { body: { fqdn } },
          { onSettled: () => resolve(), onError: () => { failed++; } },
        );
      });
    }
    if (failed === 0) {
      notifications.show({ color: "gray", message: `${fqdns.length} hosts ignored` });
    } else {
      notifications.show({ color: "orange", message: `${fqdns.length - failed} ignored, ${failed} failed` });
    }
    setSelected(new Set());
  }

  if (locked) {
    return (
      <Alert
        icon={<IconAlertCircle size={16} />}
        color="orange"
        title="Groups tab has unsaved changes"
      >
        <Stack gap="sm">
          <Text size="sm">
            Save or discard your group changes before adding hosts from suggestions.
          </Text>
          <Button size="xs" variant="outline" color="orange" onClick={onDiscardLock} w="fit-content">
            Discard group changes
          </Button>
        </Stack>
      </Alert>
    );
  }

  if (data.suggestions.length === 0 && data.ignored.length === 0) {
    return (
      <Card withBorder>
        <Stack gap="md" align="center" py="xl">
          <Text fz={48}>🔍</Text>
          <Title order={3}>Nothing to review</Title>
          <Text c="dimmed" size="sm">
            No unknown hosts in recent traffic.
          </Text>
        </Stack>
      </Card>
    );
  }

  return (
    <Stack gap="md">
      {data.suggestions.length > 0 && (
        <Card withBorder padding="md">
          <Text fw={600} mb={4}>
            Observed in recent traffic
          </Text>
          <Text size="sm" c="dimmed" mb="md">
            Hosts seen that aren't on your known list. High allowed-hit counts usually mean
            legitimate infrastructure worth promoting.
          </Text>

          {someSelected && (
            <Group gap="xs" mb="sm">
              <Badge variant="light" color="indigo" size="sm">
                {selected.size} selected
              </Badge>
              <Button
                size="xs"
                onClick={handleBulkAccept}
                disabled={ignoreSuggestion.isPending}
                loading={false}
              >
                Accept {selected.size}
              </Button>
              <Button
                size="xs"
                variant="outline"
                onClick={handleBulkIgnore}
                disabled={ignoreSuggestion.isPending}
                loading={ignoreSuggestion.isPending}
              >
                Ignore {selected.size}
              </Button>
              <Button
                size="xs"
                variant="subtle"
                color="gray"
                onClick={() => setSelected(new Set())}
              >
                Clear
              </Button>
            </Group>
          )}

          <Table.ScrollContainer minWidth={600}>
            <Table>
              <Table.Thead>
                <Table.Tr>
                  <Table.Th style={{ width: 36 }}>
                    <Checkbox
                      checked={allSelected}
                      indeterminate={someSelected && !allSelected}
                      onChange={toggleAll}
                      aria-label="Select all"
                    />
                  </Table.Th>
                  <Table.Th>Hostname</Table.Th>
                  <Table.Th>First seen</Table.Th>
                  <Table.Th>Allowed hits</Table.Th>
                  <Table.Th>Denied hits</Table.Th>
                  <Table.Th style={{ textAlign: "right" }}>Action</Table.Th>
                </Table.Tr>
              </Table.Thead>
              <Table.Tbody>
                {data.suggestions.map((s) => (
                  <Table.Tr key={s.fqdn} data-selected={selected.has(s.fqdn) || undefined}>
                    <Table.Td>
                      <Checkbox
                        checked={selected.has(s.fqdn)}
                        onChange={() => toggleOne(s.fqdn)}
                        aria-label={`Select ${s.fqdn}`}
                      />
                    </Table.Td>
                    <Table.Td>
                      <Text size="sm" fw={500} ff="monospace">
                        {s.fqdn}
                      </Text>
                    </Table.Td>
                    <Table.Td>
                      <Text size="sm" c="dimmed">
                        {formatDateTime(s.first_seen)}
                      </Text>
                    </Table.Td>
                    <Table.Td>
                      <Text
                        size="sm"
                        c={s.allowed_hits > 100 ? "green" : s.allowed_hits === 0 ? "dimmed" : undefined}
                        fw={s.allowed_hits > 100 ? 500 : 400}
                      >
                        {s.allowed_hits.toLocaleString()}
                      </Text>
                    </Table.Td>
                    <Table.Td>
                      <Text
                        size="sm"
                        c={s.denied_hits > 50 ? "orange" : s.denied_hits === 0 ? "dimmed" : undefined}
                        fw={s.denied_hits > 50 ? 500 : 400}
                      >
                        {s.denied_hits.toLocaleString()}
                      </Text>
                    </Table.Td>
                    <Table.Td>
                      <Group gap="xs" justify="flex-end">
                        <Button
                          size="xs"
                          variant="outline"
                          onClick={() => handleIgnore(s.fqdn)}
                          disabled={ignoreSuggestion.isPending}
                        >
                          Ignore
                        </Button>
                        <Button
                          size="xs"
                          onClick={() => handlePromote(s.fqdn)}
                          disabled={ignoreSuggestion.isPending}
                        >
                          Add as known
                        </Button>
                      </Group>
                    </Table.Td>
                  </Table.Tr>
                ))}
              </Table.Tbody>
            </Table>
          </Table.ScrollContainer>
        </Card>
      )}

      {data.ignored.length > 0 && (
        <Card withBorder padding="md">
          <Text fw={600} mb={4}>
            Ignored
          </Text>
          <Text size="sm" c="dimmed" mb="md">
            Won't appear in suggestions again.
          </Text>
          <Table.ScrollContainer minWidth={400}>
            <Table>
              <Table.Thead>
                <Table.Tr>
                  <Table.Th>Hostname</Table.Th>
                  <Table.Th>Ignored at</Table.Th>
                  <Table.Th />
                </Table.Tr>
              </Table.Thead>
              <Table.Tbody>
                {data.ignored.map((s) => (
                  <Table.Tr key={s.fqdn}>
                    <Table.Td>
                      <Text size="sm" ff="monospace" c="dimmed">
                        {s.fqdn}
                      </Text>
                    </Table.Td>
                    <Table.Td>
                      <Text size="sm" c="dimmed">
                        {formatDateTime(s.created_at)}
                      </Text>
                    </Table.Td>
                    <Table.Td>
                      <Group justify="flex-end">
                        <Button
                          size="xs"
                          variant="subtle"
                          onClick={() => handleUnignore(s.fqdn)}
                          disabled={unignoreSuggestion.isPending}
                        >
                          Unignore
                        </Button>
                      </Group>
                    </Table.Td>
                  </Table.Tr>
                ))}
              </Table.Tbody>
            </Table>
          </Table.ScrollContainer>
        </Card>
      )}
    </Stack>
  );
}
