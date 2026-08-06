import { Button, Checkbox, Menu, Stack, Text } from "@mantine/core";
import { IconColumns3, IconRestore } from "@tabler/icons-react";
import type { ManagedColumnMeta } from "@/hooks/useManagedColumns";

interface ColumnsMenuProps {
    /** Columns in chooser order — usually the same array passed to `useManagedColumns`. */
    meta: ManagedColumnMeta[];
    columnVisible: Map<string, boolean>;
    setColumnVisible: (accessor: string, visible: boolean) => void;
    onReset: () => void;
}

/**
 * Show/hide chooser for a table driven by `useManagedColumns`. Reorder is not
 * offered here — the header itself is the drag target — so the menu carries a
 * tip pointing at that affordance, which has no icon of its own.
 */
export function ColumnsMenu({ meta, columnVisible, setColumnVisible, onReset }: ColumnsMenuProps) {
    return (
        <Menu shadow="md" closeOnItemClick={false} position="bottom-end">
            <Menu.Target>
                <Button variant="subtle" size="compact-xs" leftSection={<IconColumns3 size={14} />}>
                    Columns
                </Button>
            </Menu.Target>
            <Menu.Dropdown>
                <Menu.Label>Columns</Menu.Label>
                <Stack gap="xs" px="sm" py={4}>
                    {meta.map((c) => (
                        <Checkbox
                            key={c.accessor}
                            size="xs"
                            label={c.label}
                            checked={c.mandatory || (columnVisible.get(c.accessor) ?? false)}
                            disabled={c.mandatory}
                            onChange={(e) => setColumnVisible(c.accessor, e.currentTarget.checked)}
                        />
                    ))}
                </Stack>
                <Menu.Divider />
                <Menu.Item leftSection={<IconRestore size={14} />} onClick={onReset}>
                    Reset columns
                </Menu.Item>
                <Menu.Divider />
                <Text size="xs" c="dimmed" px="sm" py={4}>
                    Tip: drag a column header to reorder.
                </Text>
            </Menu.Dropdown>
        </Menu>
    );
}
