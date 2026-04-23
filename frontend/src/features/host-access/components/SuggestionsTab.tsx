import {
  Button,
  Card,
  Group,
  Stack,
  Table,
  Text,
  Title,
} from "@mantine/core";
import { notifications } from "@mantine/notifications";
import type { HostSuggestionsPage } from "@/lib/api";
import { useIgnoreSuggestion } from "@/features/host-access/hooks/useIgnoreSuggestion";
import { useUnignoreSuggestion } from "@/features/host-access/hooks/useUnignoreSuggestion";
import { useCreateKnownHosts } from "@/features/host-access/hooks/useCreateKnownHosts";
import { useDateFormatter } from "@/contexts/useDateTimePrefs";
import { toErrorMessage } from "@/lib/api-client";

interface Props {
  data: HostSuggestionsPage;
}

export function SuggestionsTab({ data }: Props) {
  const formatDateTime = useDateFormatter();
  const ignoreSuggestion = useIgnoreSuggestion();
  const unignoreSuggestion = useUnignoreSuggestion();
  const createKnownHosts = useCreateKnownHosts();

  function handlePromote(fqdn: string) {
    createKnownHosts.mutate(
      { body: { fqdns: [fqdn] } },
      {
        onSuccess: () =>
          notifications.show({ color: "green", message: `${fqdn} added to known hosts` }),
        onError: (err) =>
          notifications.show({ color: "red", title: "Failed to add host", message: toErrorMessage(err) }),
      },
    );
  }

  function handleIgnore(fqdn: string) {
    ignoreSuggestion.mutate(
      { body: { fqdn } },
      {
        onSuccess: () =>
          notifications.show({ color: "gray", message: `${fqdn} ignored` }),
        onError: (err) =>
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
        onError: (err) =>
          notifications.show({ color: "red", title: "Failed to unignore", message: toErrorMessage(err) }),
      },
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
          <Table.ScrollContainer minWidth={600}>
            <Table>
              <Table.Thead>
                <Table.Tr>
                  <Table.Th>FQDN</Table.Th>
                  <Table.Th>First seen</Table.Th>
                  <Table.Th>Allowed hits</Table.Th>
                  <Table.Th>Denied hits</Table.Th>
                  <Table.Th style={{ textAlign: "right" }}>Action</Table.Th>
                </Table.Tr>
              </Table.Thead>
              <Table.Tbody>
                {data.suggestions.map((s) => (
                  <Table.Tr key={s.fqdn}>
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
                          disabled={createKnownHosts.isPending}
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
                  <Table.Th>FQDN</Table.Th>
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
