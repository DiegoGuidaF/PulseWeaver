import { Alert, Button, Stack, Text } from "@mantine/core";
import { IconAlertCircle } from "@tabler/icons-react";
import { toErrorMessage } from "@/lib/api-client/errors";

interface ErrorStateProps {
    /** Raw error from a query/mutation. The display string is extracted via `toErrorMessage`. */
    error?: unknown;
    /** Explicit message; overrides anything derived from `error`. */
    message?: string;
    /** Heading for the alert. */
    title?: string;
    /** When provided, renders a "Try again" action (e.g. a query's `refetch`). */
    onRetry?: () => void;
}

/**
 * Inline error state for failed data loads. Sibling to `EmptyState` — use it in a
 * view's `isError` branch where `ErrorBoundary` (crashes) and notifications (mutations)
 * do not apply. Mantine `Alert` provides `role="alert"` for assistive technologies.
 */
export function ErrorState({ error, message, title = "Failed to load", onRetry }: ErrorStateProps) {
    const detail = message ?? (error !== undefined ? toErrorMessage(error) : undefined);

    return (
        <Alert icon={<IconAlertCircle size={16} />} color="red" title={title}>
            <Stack gap="sm" align="flex-start">
                {detail && <Text size="sm">{detail}</Text>}
                {onRetry && (
                    <Button size="xs" variant="light" color="red" onClick={onRetry}>
                        Try again
                    </Button>
                )}
            </Stack>
        </Alert>
    );
}
