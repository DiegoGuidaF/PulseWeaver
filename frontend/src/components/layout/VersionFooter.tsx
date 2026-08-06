import { Text, Tooltip, Stack } from "@mantine/core";
import { useVersion } from "@/features/system/hooks/useVersion";

const SHORT_SHA_LENGTH = 7;

/**
 * Build identity of the running backend, shown in the sidebar footer.
 *
 * Renders nothing until the query resolves — a failed or pending version
 * lookup is not worth an error state in the app chrome.
 */
export function VersionFooter() {
    const { data } = useVersion();
    if (!data) return null;

    // Release builds are stamped from the git tag, which already carries its
    // "v" prefix; un-stamped builds report "dev". Neither wants one added.
    const hasCommit = data.commit && data.commit !== "unknown";
    const shortCommit = hasCommit ? data.commit.slice(0, SHORT_SHA_LENGTH) : null;

    const label = (
        <Stack gap={2}>
            <Text size="xs">Version {data.version}</Text>
            {hasCommit && <Text size="xs">Commit {data.commit}</Text>}
            {data.build_time && (
                <Text size="xs">Built {new Date(data.build_time).toUTCString()}</Text>
            )}
        </Stack>
    );

    return (
        <Tooltip label={label} position="top" withArrow multiline>
            <Text size="xs" c="dimmed" truncate>
                {data.version}
                {shortCommit && ` · ${shortCommit}`}
            </Text>
        </Tooltip>
    );
}
