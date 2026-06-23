import { ActionIcon, Box, Group, Tooltip } from "@mantine/core";
import { IconFilter } from "@tabler/icons-react";
import type { ReactNode } from "react";
import classes from "./AccessLogTable.module.css";

/**
 * Wraps a cell value and, when `onFilter` is given, appends a hover-revealed
 * "filter by this value" control at the cell's right edge. This keeps the
 * value's own click (navigate to the entity) distinct from filtering — the two
 * gestures live on separate affordances rather than being overloaded onto one.
 */
export function FilterableCell({
    children,
    onFilter,
    filterLabel,
}: {
    children: ReactNode;
    onFilter?: () => void;
    filterLabel: string;
}) {
    return (
        <Group gap={4} wrap="nowrap" w="100%">
            <Box flex={1} miw={0}>
                {children}
            </Box>
            {onFilter && (
                <Tooltip label={filterLabel} position="left" withArrow openDelay={400}>
                    <ActionIcon
                        size="xs"
                        variant="subtle"
                        color="gray"
                        className={classes.filterAction}
                        aria-label={filterLabel}
                        onClick={(e) => {
                            e.stopPropagation();
                            onFilter();
                        }}
                    >
                        <IconFilter size={12} />
                    </ActionIcon>
                </Tooltip>
            )}
        </Group>
    );
}
