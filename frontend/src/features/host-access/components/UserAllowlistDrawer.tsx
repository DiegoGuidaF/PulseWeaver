import { useMemo, useState } from "react";
import {
  Badge,
  Button,
  Card,
  Checkbox,
  Divider,
  Drawer,
  Group,
  Loader,
  ScrollArea,
  SegmentedControl,
  Stack,
  Switch,
  Text,
  TextInput,
} from "@mantine/core";
import { notifications } from "@mantine/notifications";
import { IconSearch, IconShieldCheck, IconShieldOff } from "@tabler/icons-react";
import { UserRole } from "@/lib/api";
import type { UserHostAccessSummary, UserHostDetails } from "@/lib/api";
import { useUserHostDetails } from "@/features/host-access/hooks/useUserHostDetails";
import { useSetUserHostGrants } from "@/features/host-access/hooks/useSetUserHostGrants";
import { GroupBadge } from "@/features/host-access/components/GroupBadge";
import { toErrorMessage } from "@/lib/api-client";

interface Props {
  user: UserHostAccessSummary | null;
  onClose: () => void;
}

interface FormProps {
  userId: number;
  details: UserHostDetails;
  userSummary: UserHostAccessSummary;
  onClose: () => void;
}

type HostFilter = "all" | "granted" | "not-granted" | "via-group";

function AllowlistForm({ userId, details, userSummary, onClose }: FormProps) {
  const setGrants = useSetUserHostGrants(userId);

  const [bypass, setBypass] = useState(details.bypass);
  const [grantedGroupIds, setGrantedGroupIds] = useState<Set<number>>(
    () => new Set(details.groups.filter((g) => g.granted).map((g) => g.id)),
  );
  const [directHostIds, setDirectHostIds] = useState<Set<number>>(
    () => new Set(details.hosts.filter((h) => h.directly_granted).map((h) => h.id)),
  );

  const [hostSearch, setHostSearch] = useState("");
  const [hostFilter, setHostFilter] = useState<HostFilter>("all");
  const [showViaGroupInAll, setShowViaGroupInAll] = useState(false);

  const viaGroupHostIds = useMemo(
    () =>
      new Set(
        details.groups
          .filter((g) => grantedGroupIds.has(g.id))
          .flatMap((g) => g.hosts.map((h) => h.id)),
      ),
    [details.groups, grantedGroupIds],
  );

  const effectiveHostCount = useMemo(
    () => details.hosts.filter((h) => directHostIds.has(h.id) || viaGroupHostIds.has(h.id)).length,
    [details.hosts, directHostIds, viaGroupHostIds],
  );

  const filteredHosts = useMemo(() => {
    const search = hostSearch.trim().toLowerCase();
    return details.hosts.filter((h) => {
      if (search && !h.fqdn.toLowerCase().includes(search)) return false;
      const isDirect = directHostIds.has(h.id);
      const isViaGroup = viaGroupHostIds.has(h.id);
      switch (hostFilter) {
        case "granted":
          return isDirect || isViaGroup;
        case "not-granted":
          return !isDirect && !isViaGroup;
        case "via-group":
          return isViaGroup;
        case "all":
          if (isViaGroup && !isDirect && !showViaGroupInAll) return false;
          return true;
      }
    });
  }, [details.hosts, hostSearch, hostFilter, directHostIds, viaGroupHostIds, showViaGroupInAll]);

  const hiddenViaGroupCount = useMemo(() => {
    if (hostFilter !== "all" || showViaGroupInAll) return 0;
    const search = hostSearch.trim().toLowerCase();
    return details.hosts.filter((h) => {
      if (search && !h.fqdn.toLowerCase().includes(search)) return false;
      return viaGroupHostIds.has(h.id) && !directHostIds.has(h.id);
    }).length;
  }, [details.hosts, hostFilter, showViaGroupInAll, hostSearch, viaGroupHostIds, directHostIds]);

  function toggleGroup(id: number) {
    setGrantedGroupIds((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  }

  function toggleHost(id: number) {
    setDirectHostIds((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  }

  function handleSave() {
    setGrants.mutate(
      {
        path: { user_id: userId },
        body: {
          bypass,
          group_ids: [...grantedGroupIds],
          host_ids: [...directHostIds],
        },
      },
      {
        onSuccess: () => {
          notifications.show({ color: "green", message: "Access updated" });
          onClose();
        },
        onError: (err) =>
          notifications.show({ color: "red", title: "Failed to save", message: toErrorMessage(err) }),
      },
    );
  }

  const isNonAdminBypass = bypass && userSummary.role === UserRole.USER;
  const totalHosts = details.hosts.length;

  return (
    <Stack gap={0} h="100%">
      <ScrollArea flex={1} mih={0} p="md">
        <Stack gap="lg">
          {/* Bypass toggle */}
          <Card
            withBorder
            style={{
              borderColor: isNonAdminBypass ? "var(--mantine-color-red-6)" : undefined,
              background: isNonAdminBypass ? "var(--mantine-color-red-light)" : undefined,
            }}
          >
            <Group justify="space-between" align="flex-start">
              <div style={{ flex: 1 }}>
                <Group gap="xs" mb={4}>
                  <Text fw={600}>Allow all hosts</Text>
                  {isNonAdminBypass && (
                    <Badge color="red" size="sm">
                      Risky
                    </Badge>
                  )}
                </Group>
                <Text size="sm" c="dimmed">
                  {bypass
                    ? "Skips the allowlist below. This user can reach every known host."
                    : "Allowlist below is enforced on every request."}
                </Text>
              </div>
              <Switch
                checked={bypass}
                onChange={(e) => setBypass(e.currentTarget.checked)}
              />
            </Group>
          </Card>

          {/* Effective access summary */}
          <div>
            <Text
              size="xs"
              c="dimmed"
              fw={600}
              tt="uppercase"
              style={{ letterSpacing: "0.05em" }}
              mb={6}
            >
              Effective access
            </Text>
            {bypass ? (
              <Group gap="xs">
                <IconShieldOff size={16} stroke={1.5} color="var(--mantine-color-dimmed)" />
                <Text size="sm">All {totalHosts} known hosts (allow-all on).</Text>
              </Group>
            ) : effectiveHostCount === 0 ? (
              <Group gap="xs">
                <IconShieldOff size={16} stroke={1.5} color="var(--mantine-color-red-6)" />
                <Text size="sm" c="red" fw={500}>
                  No hosts — all requests will be denied.
                </Text>
              </Group>
            ) : (
              <Group gap="xs">
                <IconShieldCheck size={16} stroke={1.5} color="var(--mantine-color-indigo-6)" />
                <Text size="sm">
                  <strong>{effectiveHostCount}</strong> of {totalHosts} hosts
                  {viaGroupHostIds.size > 0 && <> · {viaGroupHostIds.size} via group</>}
                  {directHostIds.size > 0 && <> · {directHostIds.size} direct</>}
                </Text>
              </Group>
            )}
          </div>

          <Divider />

          {/* Groups */}
          <div style={{ opacity: bypass ? 0.5 : 1, pointerEvents: bypass ? "none" : "auto" }}>
            <Text fw={600} mb="xs">
              Host groups
            </Text>
            {details.groups.length === 0 ? (
              <Text size="sm" c="dimmed">
                No groups configured yet.
              </Text>
            ) : (
              <Stack gap="xs">
                {details.groups.map((g) => {
                  const on = grantedGroupIds.has(g.id);
                  return (
                    <label
                      key={g.id}
                      style={{
                        display: "flex",
                        alignItems: "center",
                        gap: 12,
                        padding: 10,
                        border: on
                          ? "1px solid var(--mantine-color-indigo-3)"
                          : "1px solid var(--mantine-color-default-border)",
                        background: on ? "var(--mantine-color-indigo-light)" : "transparent",
                        borderRadius: 8,
                        cursor: "pointer",
                      }}
                    >
                      <Checkbox
                        checked={on}
                        onChange={() => toggleGroup(g.id)}
                        styles={{ input: { cursor: "pointer" } }}
                      />
                      <div style={{ flex: 1 }}>
                        <Group gap="xs">
                          <GroupBadge group={g} size="sm" />
                          <Text size="xs" c="dimmed">
                            {g.hosts.length} {g.hosts.length === 1 ? "host" : "hosts"}
                          </Text>
                        </Group>
                        {g.hosts.length > 0 && (
                          <Text size="xs" c="dimmed" ff="monospace" mt={4}>
                            {g.hosts
                              .slice(0, 4)
                              .map((h) => h.fqdn)
                              .join(" · ")}
                            {g.hosts.length > 4 && ` · +${g.hosts.length - 4} more`}
                          </Text>
                        )}
                      </div>
                    </label>
                  );
                })}
              </Stack>
            )}
          </div>

          <Divider />

          {/* Individual hosts */}
          <div style={{ opacity: bypass ? 0.5 : 1, pointerEvents: bypass ? "none" : "auto" }}>
            <Group justify="space-between" mb="xs">
              <Text fw={600}>Individual hosts</Text>
              <Text size="xs" c="dimmed">
                Grant beyond what groups cover
              </Text>
            </Group>
            {details.hosts.length === 0 ? (
              <Text size="sm" c="dimmed">
                No known hosts configured yet.
              </Text>
            ) : (
              <Stack gap="xs">
                <TextInput
                  placeholder="Search hosts…"
                  value={hostSearch}
                  onChange={(e) => setHostSearch(e.currentTarget.value)}
                  leftSection={<IconSearch size={14} stroke={1.5} />}
                  size="xs"
                />
                <SegmentedControl
                  size="xs"
                  fullWidth
                  value={hostFilter}
                  onChange={(v) => setHostFilter(v as HostFilter)}
                  data={[
                    { label: "All", value: "all" },
                    { label: "Granted", value: "granted" },
                    { label: "Not granted", value: "not-granted" },
                    { label: "Via group", value: "via-group" },
                  ]}
                />
                {filteredHosts.length === 0 ? (
                  <Text size="sm" c="dimmed" ta="center" py="md">
                    No hosts match.
                  </Text>
                ) : (
                  <Stack
                    gap={0}
                    style={{
                      border: "1px solid var(--mantine-color-default-border)",
                      borderRadius: 8,
                      overflow: "hidden",
                    }}
                  >
                    {filteredHosts.map((h, i) => {
                      const isDirect = directHostIds.has(h.id);
                      const isCoveredByGroup = viaGroupHostIds.has(h.id);
                      return (
                        <label
                          key={h.id}
                          style={{
                            display: "flex",
                            alignItems: "center",
                            gap: 12,
                            padding: "8px 12px",
                            borderBottom:
                              i < filteredHosts.length - 1
                                ? "1px solid var(--mantine-color-default-border)"
                                : "none",
                            cursor: isCoveredByGroup && !isDirect ? "default" : "pointer",
                          }}
                        >
                          <Checkbox
                            checked={isDirect || isCoveredByGroup}
                            disabled={isCoveredByGroup && !isDirect}
                            onChange={() => toggleHost(h.id)}
                            styles={{
                              input: {
                                cursor: isCoveredByGroup && !isDirect ? "default" : "pointer",
                              },
                            }}
                          />
                          <Text size="sm" ff="monospace" style={{ flex: 1 }}>
                            {h.fqdn}
                          </Text>
                          {isCoveredByGroup && (
                            <Badge variant="light" color="indigo" size="xs">
                              via group
                            </Badge>
                          )}
                          {isDirect && !isCoveredByGroup && (
                            <Badge variant="outline" color="indigo" size="xs">
                              direct
                            </Badge>
                          )}
                        </label>
                      );
                    })}
                  </Stack>
                )}
                {hiddenViaGroupCount > 0 && (
                  <Button
                    variant="subtle"
                    size="xs"
                    onClick={() => setShowViaGroupInAll(true)}
                  >
                    Show {hiddenViaGroupCount} covered by groups
                  </Button>
                )}
                {showViaGroupInAll && hostFilter === "all" && (
                  <Button
                    variant="subtle"
                    size="xs"
                    onClick={() => setShowViaGroupInAll(false)}
                  >
                    Hide via-group rows
                  </Button>
                )}
              </Stack>
            )}
          </div>
        </Stack>
      </ScrollArea>

      <Divider />
      <Group justify="flex-end" p="md" gap="xs">
        <Button variant="outline" onClick={onClose}>
          Cancel
        </Button>
        <Button onClick={handleSave} disabled={setGrants.isPending} loading={setGrants.isPending}>
          Save
        </Button>
      </Group>
    </Stack>
  );
}

export function UserAllowlistDrawer({ user, onClose }: Props) {
  const details = useUserHostDetails(user?.id ?? null);

  return (
    <Drawer
      opened={user !== null}
      onClose={onClose}
      position="right"
      size="md"
      title={
        user ? (
          <Group gap="xs">
            <Text fw={600}>{user.display_name}</Text>
            <Badge
              variant="light"
              color={
                user.role === UserRole.SUPERADMIN
                  ? "violet"
                  : user.role === UserRole.ADMIN
                    ? "indigo"
                    : "gray"
              }
              size="sm"
            >
              {user.role}
            </Badge>
          </Group>
        ) : null
      }
      styles={{
        body: {
          display: "flex",
          flexDirection: "column",
          height: "100%",
          padding: 0,
          overflow: "hidden",
        },
      }}
    >
      {details.isFetching || !details.data || !user ? (
        <Stack align="center" justify="center" h="100%" gap="xs">
          <Loader size="sm" />
          <Text size="sm" c="dimmed">
            Loading access details…
          </Text>
        </Stack>
      ) : (
        <AllowlistForm
          key={user.id}
          userId={user.id}
          details={details.data}
          userSummary={user}
          onClose={onClose}
        />
      )}
    </Drawer>
  );
}
