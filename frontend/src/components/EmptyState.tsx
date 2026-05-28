import { Stack, Text, type MantineColor } from "@mantine/core";
import type { ComponentType, ReactNode } from "react";

interface EmptyStateProps {
    icon: ComponentType<{ size?: number; stroke?: number; color?: string }>;
    title: string;
    description?: string;
    color?: MantineColor;
    /** Optional call-to-action (e.g. a Button) rendered below the description. */
    action?: ReactNode;
}

export function EmptyState({ icon: Icon, title, description, color = "dimmed", action }: EmptyStateProps) {
    return (
        <Stack align="center" justify="center" gap="xs" py="xl">
            <Icon size={40} stroke={1.5} color={`var(--mantine-color-${color}-5, var(--mantine-color-dimmed))`} />
            <Text fw={500} c="dimmed">
                {title}
            </Text>
            {description && (
                <Text size="sm" c="dimmed" ta="center" maw={400}>
                    {description}
                </Text>
            )}
            {action && <Stack mt="sm">{action}</Stack>}
        </Stack>
    );
}
