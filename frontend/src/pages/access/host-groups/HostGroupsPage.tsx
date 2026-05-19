import { useEffect, useReducer } from "react";
import { Center, Loader, Stack, Text, Title } from "@mantine/core";
import { useHostGroups } from "@/features/host-access/hooks/useHostGroups";
import { useHosts } from "@/features/host-access/hooks/useHosts";
import { HostGroupsTab } from "@/features/host-access/components/HostGroupsTab";
import { useUnsavedChangesGuard } from "@/hooks/useUnsavedChangesGuard";
import {
  groupsDraftReducer,
  initialGroupsDraft,
  isDirtyGroups,
} from "@/features/host-access/drafts/hostGroupsDraft";

export function HostGroupsPage() {
  const hostGroupsQuery = useHostGroups();
  const hostsQuery = useHosts();

  const [groupsState, groupsDispatch] = useReducer(
    groupsDraftReducer,
    undefined,
    initialGroupsDraft,
  );

  useEffect(() => {
    if (hostGroupsQuery.data) {
      groupsDispatch({ type: "reset", groups: hostGroupsQuery.data.groups });
    }
  }, [hostGroupsQuery.data]);

  const dirty = isDirtyGroups(groupsState);
  useUnsavedChangesGuard(dirty);

  const serverHosts = hostsQuery.data?.hosts ?? [];
  const loading = hostGroupsQuery.isFetching;

  return (
    <Stack maw={1100} gap="md" pb={dirty ? 80 : undefined}>
      <div>
        <Title order={1}>Host Groups</Title>
        <Text c="dimmed" mt={4}>
          Bundle hosts into groups to grant access to multiple services at once.
        </Text>
      </div>

      {loading ? (
        <Center py="xl">
          <Loader />
        </Center>
      ) : (
        <HostGroupsTab
          state={groupsState}
          dispatch={groupsDispatch}
          serverHosts={serverHosts}
        />
      )}
    </Stack>
  );
}
