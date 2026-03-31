import { Group, Text, ActionIcon } from "@mantine/core";
import { IconChevronLeft, IconChevronRight } from "@tabler/icons-react";
import { useCallback, useRef, useState } from "react";

interface CursorPaginationProps {
    /** Total number of items across all pages */
    total: number;
    /** Cursor for the next page, null if on the last page */
    nextCursor: string | null;
    /** Items per page (default 25) */
    pageSize?: number;
    /** Called when the user navigates — pass the cursor as `before_id` to the API */
    onCursorChange: (cursor: string | null) => void;
    /** When this value changes, internal page state resets to page 0 */
    resetKey?: string;
}

export function CursorPagination({
    total,
    nextCursor,
    pageSize = 25,
    onCursorChange,
    resetKey,
}: CursorPaginationProps) {
    const [page, setPage] = useState(0);
    const cursorCache = useRef<Map<number, string | null>>(new Map([[0, null]]));

    // Reset internal state when resetKey changes (e.g. filter change)
    const [prevResetKey, setPrevResetKey] = useState(resetKey);
    if (resetKey !== prevResetKey) {
        setPrevResetKey(resetKey);
        setPage(0);
        // cursorCache needs no reset: goNext always overwrites entries before
        // reading them, and goPrev is disabled when page === 0.
    }

    const totalPages = Math.max(1, Math.ceil(total / pageSize));

    const goNext = useCallback(() => {
        if (!nextCursor) return;
        const next = page + 1;
        cursorCache.current.set(next, nextCursor);
        setPage(next);
        onCursorChange(nextCursor);
    }, [nextCursor, page, onCursorChange]);

    const goPrev = useCallback(() => {
        if (page <= 0) return;
        const prev = page - 1;
        const cursor = cursorCache.current.get(prev) ?? null;
        setPage(prev);
        onCursorChange(cursor);
    }, [page, onCursorChange]);

    return (
        <Group justify="space-between" mt="md">
            <Text size="sm" c="dimmed">
                {total.toLocaleString()} {total === 1 ? "result" : "results"}
            </Text>
            <Group gap="xs">
                <ActionIcon
                    variant="subtle"
                    disabled={page <= 0}
                    onClick={goPrev}
                    aria-label="Previous page"
                >
                    <IconChevronLeft size={18} />
                </ActionIcon>
                <Text size="sm">
                    Page {page + 1} of {totalPages}
                </Text>
                <ActionIcon
                    variant="subtle"
                    disabled={!nextCursor}
                    onClick={goNext}
                    aria-label="Next page"
                >
                    <IconChevronRight size={18} />
                </ActionIcon>
            </Group>
        </Group>
    );
}
