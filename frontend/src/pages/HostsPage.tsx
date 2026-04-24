import { Stack, Tabs, Text, Title, Badge } from "@mantine/core";
import { useKnownHosts } from "@/features/host-access/hooks/useKnownHosts";
import { useHostGroups } from "@/features/host-access/hooks/useHostGroups";
import { useHostSuggestions } from "@/features/host-access/hooks/useHostSuggestions";
import { KnownHostsTab } from "@/features/host-access/components/KnownHostsTab";
import { HostGroupsTab } from "@/features/host-access/components/HostGroupsTab";
import { SuggestionsTab } from "@/features/host-access/components/SuggestionsTab";

export function HostsPage() {
  const knownHosts = useKnownHosts();
  const hostGroups = useHostGroups();
  const suggestions = useHostSuggestions();

  const hosts = knownHosts.data ?? [];
  const groups = hostGroups.data ?? [];
  const suggestionsData = suggestions.data;

  const suggestionCount = suggestionsData?.suggestions.length ?? 0;

  return (
    <Stack maw={1100} gap="md">
      <div>
        <Title order={1}>Hosts</Title>
        <Text c="dimmed" mt={4}>
          Curate which upstream services your users can reach.
        </Text>
      </div>

      <Tabs defaultValue="hosts" keepMounted={false}>
        <Tabs.List>
          <Tabs.Tab
            value="hosts"
            rightSection={
              <Badge size="xs" variant="light" color="gray">
                {hosts.length}
              </Badge>
            }
          >
            Known hosts
          </Tabs.Tab>
          <Tabs.Tab
            value="groups"
            rightSection={
              <Badge size="xs" variant="light" color="gray">
                {groups.length}
              </Badge>
            }
          >
            Groups
          </Tabs.Tab>
          <Tabs.Tab
            value="suggestions"
            rightSection={
              <Badge
                size="xs"
                variant="light"
                color={suggestionCount > 0 ? "orange" : "gray"}
              >
                {suggestionCount}
              </Badge>
            }
          >
            Suggestions
          </Tabs.Tab>
        </Tabs.List>

        <Tabs.Panel value="hosts" pt="md">
          <KnownHostsTab hosts={hosts} groups={groups} />
        </Tabs.Panel>

        <Tabs.Panel value="groups" pt="md">
          <HostGroupsTab groups={groups} hosts={hosts} />
        </Tabs.Panel>

        <Tabs.Panel value="suggestions" pt="md">
          {suggestionsData ? (
            <SuggestionsTab data={suggestionsData} />
          ) : (
            <Text c="dimmed" size="sm">
              Loading…
            </Text>
          )}
        </Tabs.Panel>
      </Tabs>
    </Stack>
  );
}
