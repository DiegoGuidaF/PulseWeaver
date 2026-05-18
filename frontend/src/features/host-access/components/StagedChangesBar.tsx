import { Affix, Button, Group, Paper, Stack, Text, Transition } from "@mantine/core";

export const STAGED_BAR_HEIGHT = 80;
import { IconAlertCircle } from "@tabler/icons-react";

interface Props {
  visible: boolean;
  summary: string;
  detail?: string | null;
  saving?: boolean;
  onSave: () => void;
  onDiscard: () => void;
}

export function StagedChangesBar({
  visible,
  summary,
  detail,
  saving,
  onSave,
  onDiscard,
}: Props) {
  return (
    <Affix position={{ bottom: 20, left: 20, right: 20 }} zIndex={300}>
      <Transition mounted={visible} transition="slide-up" duration={200}>
        {(styles) => (
          <Paper
            withBorder
            shadow="md"
            radius="md"
            p="sm"
            style={{ ...styles, maxWidth: 1100, marginInline: "auto" }}
          >
            <Group justify="space-between" wrap="nowrap" gap="md">
              <Group gap="sm" wrap="nowrap">
                <IconAlertCircle
                  size={20}
                  stroke={1.5}
                  color="var(--mantine-color-orange-6)"
                />
                <Stack gap={0}>
                  <Text size="sm" fw={500}>
                    {summary}
                  </Text>
                  {detail && (
                    <Text size="xs" c="dimmed">
                      {detail}
                    </Text>
                  )}
                </Stack>
              </Group>
              <Group gap="xs" wrap="nowrap">
                <Button variant="outline" size="xs" onClick={onDiscard} disabled={saving}>
                  Discard
                </Button>
                <Button size="xs" onClick={onSave} loading={saving}>
                  Save changes
                </Button>
              </Group>
            </Group>
          </Paper>
        )}
      </Transition>
    </Affix>
  );
}
