import { Affix, Button, Checkbox, Group, Paper, Stack, Text, Transition } from "@mantine/core";

export const STAGED_BAR_HEIGHT = 80;
import { IconAlertCircle, IconAlertTriangle } from "@tabler/icons-react";

interface Props {
  visible: boolean;
  summary: string;
  detail?: string | null;
  saving?: boolean;
  onSave: () => void;
  onDiscard: () => void;
  /**
   * Switches the bar to a danger-toned variant for changes with a large blast
   * radius (e.g. enabling bypass). Pairs with `acknowledge` to require an
   * explicit confirmation step before "Save changes" activates — the bar
   * itself is the gate, so no separate confirm modal is needed (a per-toggle
   * modal would clash with the staged-save model).
   */
  warning?: {
    detail: string;
    acknowledgeLabel: string;
    acknowledged: boolean;
    onAcknowledgeChange: (acknowledged: boolean) => void;
  };
}

export function StagedChangesBar({
  visible,
  summary,
  detail,
  saving,
  onSave,
  onDiscard,
  warning,
}: Props) {
  const isWarning = warning != null;
  const saveDisabled = isWarning && !warning.acknowledged;

  return (
    <Affix position={{ bottom: 20, left: 20, right: 20 }} zIndex={300}>
      <Transition mounted={visible} transition="slide-up" duration={200}>
        {(styles) => (
          <Paper
            withBorder
            shadow="md"
            radius="md"
            p="sm"
            style={{
              ...styles,
              maxWidth: 1100,
              marginInline: "auto",
              borderColor: isWarning ? "var(--mantine-color-yellow-6)" : undefined,
            }}
          >
            <Stack gap="sm">
              <Group justify="space-between" wrap="nowrap" gap="md" align="flex-start">
                <Group gap="sm" wrap="nowrap" align="flex-start">
                  {isWarning ? (
                    <IconAlertTriangle
                      size={20}
                      stroke={1.5}
                      color="var(--mantine-color-yellow-6)"
                    />
                  ) : (
                    <IconAlertCircle
                      size={20}
                      stroke={1.5}
                      color="var(--mantine-color-orange-6)"
                    />
                  )}
                  <Stack gap={0}>
                    <Text size="sm" fw={500}>
                      {summary}
                    </Text>
                    {isWarning ? (
                      <Text size="xs" c="yellow.6" fw={500}>
                        {warning.detail}
                      </Text>
                    ) : (
                      detail && (
                        <Text size="xs" c="dimmed">
                          {detail}
                        </Text>
                      )
                    )}
                  </Stack>
                </Group>
                <Group gap="xs" wrap="nowrap">
                  <Button variant="outline" size="xs" onClick={onDiscard} disabled={saving}>
                    Discard
                  </Button>
                  <Button size="xs" onClick={onSave} loading={saving} disabled={saveDisabled}>
                    Save changes
                  </Button>
                </Group>
              </Group>

              {isWarning && (
                <Checkbox
                  size="sm"
                  color="yellow"
                  label={warning.acknowledgeLabel}
                  checked={warning.acknowledged}
                  onChange={(e) => warning.onAcknowledgeChange(e.currentTarget.checked)}
                  disabled={saving}
                />
              )}
            </Stack>
          </Paper>
        )}
      </Transition>
    </Affix>
  );
}
