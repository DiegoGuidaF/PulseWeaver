import { Group, Text } from "@mantine/core";
import type { ReactNode } from "react";

interface PageToolbarProps {
    subtitle?: string;
    right?: ReactNode;
}

export function PageToolbar({ subtitle, right }: PageToolbarProps) {
    return (
        <Group justify="space-between" align="center" wrap="wrap">
            <Group gap="md" align="center">
                {subtitle && (
                    <Text c="dimmed" size="sm">
                        {subtitle}
                    </Text>
                )}
            </Group>
            {right && (
                <Group gap="md" align="center">
                    {right}
                </Group>
            )}
        </Group>
    );
}
