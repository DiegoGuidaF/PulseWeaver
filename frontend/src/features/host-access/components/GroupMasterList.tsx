import React, { useMemo, useState } from "react";
import {
  ActionIcon,
  Badge,
  Box,
  Group,
  Paper,
  ScrollArea,
  Stack,
  Text,
  TextInput,
  ThemeIcon,
  UnstyledButton,
} from "@mantine/core";
import { IconPlus, IconSearch } from "@tabler/icons-react";
import type { MantineColor } from "@mantine/core";
import type {
  DraftGroup,
  DraftGroupId,
  GroupsDiff,
} from "@/features/host-access/drafts/hostGroupsDraft";
import { groupColor } from "@/features/host-access/utils/groupColor";
import { resolveHostIcon } from "@/features/host-access/hostIconConfig";

interface Props {
  groups: DraftGroup[];
  tombstoned: DraftGroup[];
  selectedId: DraftGroupId | null;
  diff: GroupsDiff;
  onSelect: (id: DraftGroupId) => void;
  onCreate: () => void;
}

export function GroupMasterList({ groups, tombstoned, selectedId, diff, onSelect, onCreate }: Props) {
  const [search, setSearch] = useState("");
  const term = search.toLowerCase().trim();

  const filtered = useMemo(() => {
    if (!term) return groups;
    return groups.filter((g) => g.name.toLowerCase().includes(term));
  }, [groups, term]);

  const filteredTombstoned = useMemo(() => {
    if (!term) return tombstoned;
    return tombstoned.filter((g) => g.name.toLowerCase().includes(term));
  }, [tombstoned, term]);

  return (
    <Paper withBorder radius="md" p="xs" h="100%">
      <Stack gap="xs" h="100%">
        <Group gap="xs" wrap="nowrap">
          <TextInput
            placeholder="Search groups…"
            value={search}
            onChange={(e) => setSearch(e.currentTarget.value)}
            leftSection={<IconSearch size={14} />}
            size="xs"
            style={{ flex: 1 }}
          />
          <ActionIcon
            variant="filled"
            size="lg"
            onClick={onCreate}
            aria-label="New group"
          >
            <IconPlus size={16} />
          </ActionIcon>
        </Group>
        <ScrollArea.Autosize mah={520} type="auto">
          <Stack gap={4}>
            {filtered.length === 0 && filteredTombstoned.length === 0 ? (
              <Text size="sm" c="dimmed" ta="center" py="md">
                No groups match.
              </Text>
            ) : (
              <>
                {filtered.map((g) => (
                  <GroupRow
                    key={String(g.id)}
                    group={g}
                    selected={selectedId === g.id}
                    isDirty={diff.byId.has(g.id)}
                    isNew={diff.byId.get(g.id) === "added"}
                    onClick={() => onSelect(g.id)}
                  />
                ))}
                {filteredTombstoned.map((g) => (
                  <GroupRow
                    key={String(g.id)}
                    group={g}
                    selected={selectedId === g.id}
                    isDirty={false}
                    isNew={false}
                    isTombstoned
                    onClick={() => onSelect(g.id)}
                  />
                ))}
              </>
            )}
          </Stack>
        </ScrollArea.Autosize>
      </Stack>
    </Paper>
  );
}

interface RowProps {
  group: DraftGroup;
  selected: boolean;
  isDirty: boolean;
  isNew: boolean;
  isTombstoned?: boolean;
  onClick: () => void;
}

function GroupRow({ group, selected, isDirty, isNew, isTombstoned, onClick }: RowProps) {
  const color: MantineColor = group.color ?? groupColor(group.name);
  const resolved = resolveHostIcon(group.icon);
  return (
    <UnstyledButton
      onClick={onClick}
      style={{
        padding: "8px 10px",
        borderRadius: 6,
        opacity: isTombstoned ? 0.55 : undefined,
        background: selected ? "var(--mantine-color-default-hover)" : undefined,
        border: selected
          ? "1px solid var(--mantine-color-default-border)"
          : "1px solid transparent",
      }}
    >
      <Group gap="sm" wrap="nowrap" justify="space-between">
        <Group gap="sm" wrap="nowrap" style={{ minWidth: 0 }}>
          <ThemeIcon variant="light" color={isTombstoned ? "red" : color} size="md" radius="md">
            {resolved.kind === "tabler" ? (
              React.createElement(resolved.icon, { size: 14, stroke: 1.5 })
            ) : (
              <Text size="sm" span>
                {resolved.value}
              </Text>
            )}
          </ThemeIcon>
          <Box style={{ minWidth: 0 }}>
            <Group gap={4} wrap="nowrap">
              <Text size="sm" fw={600} truncate td={isTombstoned ? "line-through" : undefined}>
                {group.name || <Text span c="dimmed" inherit>Unnamed group</Text>}
              </Text>
              {isDirty && !isTombstoned && (
                <Box
                  aria-label="unsaved changes"
                  style={{
                    width: 6,
                    height: 6,
                    borderRadius: "50%",
                    background: "var(--mantine-color-orange-6)",
                  }}
                />
              )}
            </Group>
            {isNew ? (
              <Text size="xs" c="dimmed">
                New group
              </Text>
            ) : null}
          </Box>
        </Group>
        {isTombstoned ? (
          <Badge size="xs" variant="light" color="red">
            Will delete
          </Badge>
        ) : (
          <Badge size="xs" variant="light" color="gray">
            {group.hostIds.length}
          </Badge>
        )}
      </Group>
    </UnstyledButton>
  );
}
