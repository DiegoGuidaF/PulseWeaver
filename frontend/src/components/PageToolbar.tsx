import { Group, Text } from "@mantine/core";
import type { ReactNode } from "react";

interface PageToolbarProps {
    subtitle?: string;
    left?: ReactNode;
    right?: ReactNode;
}

export function PageToolbar({ subtitle, left, right }: PageToolbarProps) {
    return (
        <Group justify="space-between" align="center" wrap="wrap">
            <Group gap="md" align="center">
                {subtitle && (
                    <Text c="dimmed" size="sm">
                        {subtitle}
                    </Text>
                )}
                {left}
            </Group>
            {right && (
                <Group gap="md" align="center">
                    {right}
                </Group>
            )}
        </Group>
    );
}
