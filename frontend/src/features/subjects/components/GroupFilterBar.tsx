import { useState } from "react";
import { Badge, Group, Menu, Text, UnstyledButton } from "@mantine/core";
import { IconChevronDown } from "@tabler/icons-react";
import type { GroupRef } from "@/lib/api";

interface Props {
  availableGroups: GroupRef[];
  selected: Set<number>;
  onChange: (next: Set<number>) => void;
}

export function GroupFilterBar({ availableGroups, selected, onChange }: Props) {
  const [menuOpen, setMenuOpen] = useState(false);

  const selectedGroups = availableGroups.filter((g) => selected.has(g.id));
  const unselectedGroups = availableGroups.filter((g) => !selected.has(g.id));

  function removeGroup(id: number) {
    const next = new Set(selected);
    next.delete(id);
    onChange(next);
  }

  function addGroup(id: number) {
    const next = new Set(selected);
    next.add(id);
    onChange(next);
    setMenuOpen(false);
  }

  return (
    <Group gap="xs" wrap="wrap">
      {selectedGroups.map((g) => (
        <Badge
          key={g.id}
          variant="filled"
          color="indigo"
          size="sm"
          rightSection={
            <UnstyledButton
              onClick={() => removeGroup(g.id)}
              style={{ display: "flex", alignItems: "center", marginLeft: 2 }}
              aria-label={`Remove ${g.name} filter`}
            >
              <Text size="xs" lh={1}>×</Text>
            </UnstyledButton>
          }
        >
          {g.name}
        </Badge>
      ))}

      {(unselectedGroups.length > 0 || selected.size === 0) && (
        <Menu opened={menuOpen} onChange={setMenuOpen} position="bottom-start">
          <Menu.Target>
            <Badge
              variant="outline"
              color="gray"
              size="sm"
              style={{ cursor: "pointer" }}
              rightSection={<IconChevronDown size={10} />}
            >
              {selected.size === 0 ? "Group" : "+ group"}
            </Badge>
          </Menu.Target>
          {unselectedGroups.length > 0 && (
            <Menu.Dropdown>
              {unselectedGroups.map((g) => (
                <Menu.Item key={g.id} onClick={() => addGroup(g.id)}>
                  {g.name}
                </Menu.Item>
              ))}
            </Menu.Dropdown>
          )}
        </Menu>
      )}

      {selected.size > 0 && (
        <UnstyledButton
          onClick={() => onChange(new Set())}
          style={{ fontSize: 12, color: "var(--mantine-color-dimmed)" }}
        >
          Clear
        </UnstyledButton>
      )}
    </Group>
  );
}
