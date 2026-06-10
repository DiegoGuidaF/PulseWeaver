import { Group, UnstyledButton } from "@mantine/core";
import { GroupChipPicker } from "@/features/host-access/components/GroupChipPicker";

/** Sentinel id for the "no group assigned" pseudo-entry; real group ids are always positive. */
export const UNGROUPED_GROUP_ID = -1;

const UNGROUPED_ENTRY: FilterableGroup = { id: UNGROUPED_GROUP_ID, name: "Ungrouped" };

interface FilterableGroup {
  id: number;
  name: string;
  color?: string | null;
  icon?: string | null;
}

interface Props {
  availableGroups: FilterableGroup[];
  selected: Set<number>;
  onChange: (next: Set<number>) => void;
  /** Adds an "Ungrouped" entry (id UNGROUPED_GROUP_ID) for filtering items with no group assigned. */
  showUngrouped?: boolean;
}

export function GroupFilterBar({ availableGroups, selected, onChange, showUngrouped }: Props) {
  const entries = showUngrouped ? [...availableGroups, UNGROUPED_ENTRY] : availableGroups;

  return (
    <Group gap="xs" wrap="wrap">
      <GroupChipPicker
        availableGroups={entries}
        selected={selected}
        onChange={onChange}
        emptyLabel="Group"
        addLabel="+ group"
        removeAriaLabel={(name) => `Remove ${name} filter`}
      />

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
