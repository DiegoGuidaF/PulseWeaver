import { useEffect, useMemo, useReducer, useState } from "react";
import {
  Badge,
  Button,
  Center,
  Collapse,
  Group,
  Loader,
  Stack,
  Text,
  Title,
  UnstyledButton,
} from "@mantine/core";
import { IconChevronDown, IconChevronRight, IconRefresh } from "@tabler/icons-react";
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
  const [suggestionsOpen, setSuggestionsOpen] = useState(true);

  useEffect(() => {
    if (hostsQuery.data) hostsDispatch({ type: "reset", hosts: hostsQuery.data.hosts });
  }, [hostsQuery.data]);

  const dirty = isDirtyHosts(hostsState);
  useUnsavedChangesGuard(dirty);

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

  // Sections gate on isPending (initial load only) so the table is not replaced by a
  // spinner on every background refetch; the suggestions count badge uses isFetching as a
  // subtle refetch indicator.
  const hostsLoading = hostsQuery.isPending;
  const suggestionsLoading = suggestions.isPending;
  const suggestionsFetching = suggestions.isFetching;

  return (
    <Stack maw={1100} gap="md" pb={dirty ? 80 : undefined}>
      <div>
        <Title order={1}>Hosts</Title>
        <Text c="dimmed" mt={4}>
          Curate which upstream services your users can reach.
        </Text>
      </div>

      {/* Known hosts — the staged catalog, saved via the bottom bar */}
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
        <HostsTab state={hostsState} dispatch={hostsDispatch} serverGroups={groups} />
      )}

      {/* Observed in recent traffic — promoting a suggestion stages a host into the
          same draft above; nothing is committed until Save. */}
      <div>
        <Group justify="space-between" wrap="wrap" gap="xs" align="center">
          <UnstyledButton
            onClick={() => setSuggestionsOpen((open) => !open)}
            aria-expanded={suggestionsOpen}
          >
            <Group gap="xs" wrap="nowrap">
              {suggestionsOpen ? (
                <IconChevronDown size={16} />
              ) : (
                <IconChevronRight size={16} />
              )}
              <Text fw={600}>Observed in recent traffic</Text>
              {suggestionsFetching ? (
                <Loader size="xs" type="dots" />
              ) : (
                <Badge size="sm" variant="light" color={suggestionCount > 0 ? "orange" : "gray"}>
                  {suggestionCount}
                </Badge>
              )}
            </Group>
          </UnstyledButton>
          <Button
            size="xs"
            variant="subtle"
            leftSection={<IconRefresh size={14} />}
            onClick={() => suggestions.refetch()}
          >
            Refresh
          </Button>
        </Group>
        <Text size="xs" c="dimmed" mt={4} ml={24}>
          Hosts seen in recent traffic that aren't on your known list. Promoting one stages
          a host above — nothing is granted until you Save.
        </Text>

        <Collapse expanded={suggestionsOpen}>
          <Stack gap="md" mt="md">
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
          </Stack>
        </Collapse>
      </div>
    </Stack>
  );
}
