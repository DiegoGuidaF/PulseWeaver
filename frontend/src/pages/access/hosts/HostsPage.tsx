import { useEffect, useMemo, useReducer } from "react";
import { Center, Loader, Stack, Tabs, Text, Title, Badge } from "@mantine/core";
import { useHosts } from "@/features/host-access/hooks/useHosts";
import { useHostGroups } from "@/features/host-access/hooks/useHostGroups";
import { useHostSuggestions } from "@/features/host-access/hooks/useHostSuggestions";
import { HostsTab } from "@/features/host-access/components/HostsTab";
import { SuggestionsTab } from "@/features/host-access/components/SuggestionsTab";
import { ErrorState } from "@/components/ErrorState";
import { useUnsavedChangesGuard } from "@/hooks/useUnsavedChangesGuard";
import {
  hostsDraftReducer,
  initialHostsDraft,
  isDirtyHosts,
} from "@/features/host-access/drafts/knownHostsDraft";

export function HostsPage() {
  const hostsQuery = useHosts();
  const hostGroupsQuery = useHostGroups();
  const suggestions = useHostSuggestions();

  const [hostsState, hostsDispatch] = useReducer(hostsDraftReducer, undefined, initialHostsDraft);

  useEffect(() => {
    if (hostsQuery.data) hostsDispatch({ type: "reset", hosts: hostsQuery.data.hosts });
  }, [hostsQuery.data]);

  const dirty = isDirtyHosts(hostsState);
  useUnsavedChangesGuard(dirty);

  const hosts = hostsQuery.data?.hosts ?? [];
  const groups = hostGroupsQuery.data?.groups ?? [];

  const draftFqdns = useMemo(
    () => new Set(Array.from(hostsState.draft.values()).map((h) => h.fqdn)),
    [hostsState.draft],
  );
  const suggestionsData = useMemo(() => {
    if (!suggestions.data) return undefined;
    return {
      ...suggestions.data,
      suggestions: suggestions.data.suggestions.filter((s) => !draftFqdns.has(s.fqdn)),
    };
  }, [suggestions.data, draftFqdns]);
  const suggestionCount = suggestionsData?.suggestions.length ?? 0;

  // Panels gate on isPending (initial load only) so the table is not replaced by a
  // spinner on every background refetch; the tab badges use isFetching as a subtle
  // refetch indicator.
  const hostsLoading = hostsQuery.isPending;
  const suggestionsLoading = suggestions.isPending;
  const hostsFetching = hostsQuery.isFetching;
  const suggestionsFetching = suggestions.isFetching;

  return (
    <Stack maw={1100} gap="md" pb={dirty ? 80 : undefined}>
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
              hostsFetching ? (
                <Loader size="xs" type="dots" />
              ) : (
                <Badge size="xs" variant="light" color={isDirtyHosts(hostsState) ? "orange" : "gray"}>
                  {hosts.length}
                </Badge>
              )
            }
          >
            Hosts
          </Tabs.Tab>
          <Tabs.Tab
            value="suggestions"
            rightSection={
              suggestionsFetching ? (
                <Loader size="xs" type="dots" />
              ) : (
                <Badge
                  size="xs"
                  variant="light"
                  color={suggestionCount > 0 ? "orange" : "gray"}
                >
                  {suggestionCount}
                </Badge>
              )
            }
          >
            Suggestions
          </Tabs.Tab>
        </Tabs.List>

        <Tabs.Panel value="hosts" pt="md">
          {hostsQuery.isError ? (
            <ErrorState
              error={hostsQuery.error}
              title="Failed to load hosts"
              onRetry={() => hostsQuery.refetch()}
            />
          ) : hostsLoading ? (
            <Center py="xl">
              <Loader />
            </Center>
          ) : (
            <HostsTab
              state={hostsState}
              dispatch={hostsDispatch}
              serverGroups={groups}
            />
          )}
        </Tabs.Panel>

        <Tabs.Panel value="suggestions" pt="md">
          {suggestions.isError ? (
            <ErrorState
              error={suggestions.error}
              title="Failed to load suggestions"
              onRetry={() => suggestions.refetch()}
            />
          ) : suggestionsLoading ? (
            <Center py="xl">
              <Loader />
            </Center>
          ) : suggestionsData ? (
            <SuggestionsTab
              data={suggestionsData}
              locked={false}
              onDiscardLock={() => {}}
              onRefresh={() => suggestions.refetch()}
              onStageHosts={(fqdns) => {
                fqdns.forEach((fqdn) => {
                  const id: `new-${string}` = `new-${crypto.randomUUID()}`;
                  hostsDispatch({ type: "add", id, host: { fqdn, groupIds: [], source: "suggestion" } });
                });
              }}
            />
          ) : (
            <ErrorState title="Failed to load suggestions" onRetry={() => suggestions.refetch()} />
          )}
        </Tabs.Panel>
      </Tabs>
    </Stack>
  );
}
