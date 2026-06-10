import { useState } from "react";
import { Badge, Menu, Text, UnstyledButton } from "@mantine/core";
import { IconChevronDown } from "@tabler/icons-react";
import { GroupBadge } from "@/features/host-access/components/GroupBadge";

interface PickableGroup {
  id: number;
  name: string;
  color?: string | null;
  icon?: string | null;
}

interface Props {
  availableGroups: PickableGroup[];
  selected: Set<number>;
  onChange: (next: Set<number>) => void;
  /** Label on the add-chip while nothing is selected yet. */
  emptyLabel?: string;
  /** Label on the add-chip once at least one group is selected. */
  addLabel?: string;
  /** Builds the aria-label for a selected chip's remove button, e.g. `Remove ${name} filter` or `Unassign ${name}`. */
  removeAriaLabel: (name: string) => string;
}

/** Chip-based multi-select: selected groups render as removable `GroupBadge` chips with their real color/icon, plus a dropdown to add more. Renders bare elements — wrap in a `Group` to lay them out. */
export function GroupChipPicker({
  availableGroups,
  selected,
  onChange,
  emptyLabel = "Group",
  addLabel = "+ group",
  removeAriaLabel,
}: Props) {
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
    <>
      {selectedGroups.map((g) => (
        <GroupBadge
          key={g.id}
          group={g}
          size="sm"
          rightSection={
            <UnstyledButton
              onClick={() => removeGroup(g.id)}
              style={{ display: "flex", alignItems: "center", marginLeft: 2 }}
              aria-label={removeAriaLabel(g.name)}
            >
              <Text size="xs" lh={1}>×</Text>
            </UnstyledButton>
          }
        />
      ))}

      {(unselectedGroups.length > 0 || selected.size === 0) && (
        <Menu opened={menuOpen} onChange={setMenuOpen} position="bottom-start">
          <Menu.Target>
            {/* Badge rendered as <button> so Menu.Target's aria-haspopup/aria-expanded are valid */}
            <Badge
              component="button"
              type="button"
              variant="outline"
              color="gray"
              size="sm"
              style={{ cursor: "pointer", minHeight: 24 }}
              rightSection={<IconChevronDown size={10} />}
            >
              {selected.size === 0 ? emptyLabel : addLabel}
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
    </>
  );
}
