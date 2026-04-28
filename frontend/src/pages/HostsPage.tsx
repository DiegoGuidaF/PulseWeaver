import { useEffect, useMemo, useReducer } from "react";
import { Center, Loader, Stack, Tabs, Text, Title, Badge } from "@mantine/core";
import { useKnownHosts } from "@/features/host-access/hooks/useKnownHosts";
import { useHostGroups } from "@/features/host-access/hooks/useHostGroups";
import { useHostSuggestions } from "@/features/host-access/hooks/useHostSuggestions";
import { KnownHostsTab } from "@/features/host-access/components/KnownHostsTab";
import { HostGroupsTab } from "@/features/host-access/components/HostGroupsTab";
import { SuggestionsTab } from "@/features/host-access/components/SuggestionsTab";
import { useUnsavedChangesGuard } from "@/hooks/useUnsavedChangesGuard";
import {
  hostsDraftReducer,
  initialHostsDraft,
  isDirtyHosts,
} from "@/features/host-access/drafts/knownHostsDraft";
import {
  groupsDraftReducer,
  initialGroupsDraft,
  isDirtyGroups,
} from "@/features/host-access/drafts/hostGroupsDraft";

export function HostsPage() {
  const knownHosts = useKnownHosts();
  const hostGroups = useHostGroups();
  const suggestions = useHostSuggestions();

  const [hostsState, hostsDispatch] = useReducer(hostsDraftReducer, undefined, initialHostsDraft);
  const [groupsState, groupsDispatch] = useReducer(
    groupsDraftReducer,
    undefined,
    initialGroupsDraft,
  );

  // Server data → draft sync. Reset whenever server identity changes; this races with
  // user edits intentionally — the leave guard plus explicit save/discard cover the
  // intended flows. Background refetches arriving while the user has dirty drafts will
  // overwrite them, which we accept as a rare edge case.
  useEffect(() => {
    if (knownHosts.data) hostsDispatch({ type: "reset", hosts: knownHosts.data });
  }, [knownHosts.data]);

  useEffect(() => {
    if (hostGroups.data) groupsDispatch({ type: "reset", groups: hostGroups.data });
  }, [hostGroups.data]);

  const dirty = isDirtyHosts(hostsState) || isDirtyGroups(groupsState);
  useUnsavedChangesGuard(dirty);

  const hosts = knownHosts.data ?? [];
  const groups = hostGroups.data ?? [];

  // Exclude any fqdn already staged in the hosts draft so that cache refetches
  // don't bring suggestions back for hosts the user has already accepted.
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

  const hostsLoading = knownHosts.isPending;
  const groupsLoading = hostGroups.isPending;
  const suggestionsLoading = suggestions.isPending;

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
              hostsLoading ? (
                <Loader size="xs" type="dots" />
              ) : (
                <Badge size="xs" variant="light" color={isDirtyHosts(hostsState) ? "orange" : "gray"}>
                  {hosts.length}
                </Badge>
              )
            }
          >
            Known hosts
          </Tabs.Tab>
          <Tabs.Tab
            value="groups"
            rightSection={
              groupsLoading ? (
                <Loader size="xs" type="dots" />
              ) : (
                <Badge
                  size="xs"
                  variant="light"
                  color={isDirtyGroups(groupsState) ? "orange" : "gray"}
                >
                  {groups.length}
                </Badge>
              )
            }
          >
            Groups
          </Tabs.Tab>
          <Tabs.Tab
            value="suggestions"
            rightSection={
              suggestionsLoading ? (
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
          {hostsLoading ? (
            <Center py="xl">
              <Loader />
            </Center>
          ) : (
            <KnownHostsTab
              state={hostsState}
              dispatch={hostsDispatch}
              serverGroups={groups}
              locked={isDirtyGroups(groupsState)}
              onDiscardLock={() => groupsDispatch({ type: "discard" })}
            />
          )}
        </Tabs.Panel>

        <Tabs.Panel value="groups" pt="md">
          {groupsLoading ? (
            <Center py="xl">
              <Loader />
            </Center>
          ) : (
            <HostGroupsTab
              state={groupsState}
              dispatch={groupsDispatch}
              hostsState={hostsState}
              locked={isDirtyHosts(hostsState)}
              onDiscardLock={() => hostsDispatch({ type: "discard" })}
            />
          )}
        </Tabs.Panel>

        <Tabs.Panel value="suggestions" pt="md">
          {suggestionsLoading ? (
            <Center py="xl">
              <Loader />
            </Center>
          ) : suggestionsData ? (
            <SuggestionsTab
              data={suggestionsData}
              locked={isDirtyGroups(groupsState)}
              onDiscardLock={() => groupsDispatch({ type: "discard" })}
              onStageHosts={(fqdns) => {
                fqdns.forEach((fqdn) => {
                  const id: `new-${string}` = `new-${crypto.randomUUID()}`;
                  hostsDispatch({ type: "add", id, host: { fqdn, icon: null, groupIds: [], source: "suggestion" } });
                });
              }}
            />
          ) : (
            <Text c="dimmed" size="sm">
              Failed to load suggestions.
            </Text>
          )}
        </Tabs.Panel>
      </Tabs>
    </Stack>
  );
}
