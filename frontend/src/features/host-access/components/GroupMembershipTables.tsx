import React, { useMemo, useState } from "react";
import {
  Checkbox,
  Group,
  Paper,
  ScrollArea,
  Stack,
  Text,
  TextInput,
  ThemeIcon,
  Title,
} from "@mantine/core";
import { IconSearch } from "@tabler/icons-react";
import type { Id } from "@/lib/api";
import type { DraftHost } from "@/features/host-access/drafts/knownHostsDraft";
import { resolveHostIcon } from "@/features/host-access/hostIconConfig";

interface Props {
  hosts: DraftHost[];
  inGroupIds: Set<Id>;
  onToggle: (hostId: Id) => void;
  disabled?: boolean;
}

export function GroupMembershipTables({ hosts, inGroupIds, onToggle, disabled }: Props) {
  const [search, setSearch] = useState("");
  const term = search.toLowerCase().trim();

  const { inside, outside } = useMemo(() => {
    const inside: DraftHost[] = [];
    const outside: DraftHost[] = [];
    for (const h of hosts) {
      if (typeof h.id !== "number") continue;
      if (term && !h.fqdn.toLowerCase().includes(term)) continue;
      if (inGroupIds.has(h.id)) inside.push(h);
      else outside.push(h);
    }
    inside.sort((a, b) => a.fqdn.localeCompare(b.fqdn));
    outside.sort((a, b) => a.fqdn.localeCompare(b.fqdn));
    return { inside, outside };
  }, [hosts, inGroupIds, term]);

  return (
    <Stack gap="sm">
      <TextInput
        placeholder="Search hosts…"
        value={search}
        onChange={(e) => setSearch(e.currentTarget.value)}
        leftSection={<IconSearch size={14} />}
        size="xs"
      />
      <HostList
        title={`In this group (${inside.length})`}
        hosts={inside}
        checked
        emptyText="No hosts in this group yet."
        onToggle={onToggle}
        disabled={disabled}
      />
      <HostList
        title={`Available hosts (${outside.length})`}
        hosts={outside}
        checked={false}
        emptyText={hosts.length === 0 ? "No known hosts." : "All matching hosts are in this group."}
        onToggle={onToggle}
        disabled={disabled}
      />
    </Stack>
  );
}

interface HostListProps {
  title: string;
  hosts: DraftHost[];
  checked: boolean;
  emptyText: string;
  onToggle: (id: Id) => void;
  disabled?: boolean;
}

function HostList({ title, hosts, checked, emptyText, onToggle, disabled }: HostListProps) {
  return (
    <Paper withBorder radius="md" p={0}>
      <Group p="sm" justify="space-between">
        <Title order={6} fw={600}>
          {title}
        </Title>
      </Group>
      {hosts.length === 0 ? (
        <Text size="sm" c="dimmed" px="sm" pb="sm">
          {emptyText}
        </Text>
      ) : (
        <ScrollArea.Autosize mah={220}>
          <Stack gap={0}>
            {hosts.map((h, i) => {
              const resolved = resolveHostIcon(h.icon);
              return (
                <Group
                  key={h.id}
                  px="sm"
                  py={6}
                  gap="sm"
                  wrap="nowrap"
                  style={{
                    borderTop:
                      i === 0 ? "1px solid var(--mantine-color-default-border)" : undefined,
                    borderBottom:
                      i < hosts.length - 1
                        ? "1px solid var(--mantine-color-default-border)"
                        : undefined,
                  }}
                >
                  <Checkbox
                    checked={checked}
                    onChange={() => onToggle(h.id as Id)}
                    disabled={disabled}
                    aria-label={`Toggle ${h.fqdn} in group`}
                  />
                  <ThemeIcon variant="subtle" color="gray" size="sm">
                    {resolved.kind === "tabler" ? (
                      React.createElement(resolved.icon, { size: 14, stroke: 1.5 })
                    ) : (
                      <Text size="sm" span>
                        {resolved.value}
                      </Text>
                    )}
                  </ThemeIcon>
                  <Text size="sm" ff="monospace">
                    {h.fqdn}
                  </Text>
                </Group>
              );
            })}
          </Stack>
        </ScrollArea.Autosize>
      )}
    </Paper>
  );
}
