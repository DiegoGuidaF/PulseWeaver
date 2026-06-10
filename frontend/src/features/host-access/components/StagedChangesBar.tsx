import type { CSSProperties } from "react";
import { Affix, Button, Checkbox, Group, Paper, Stack, Text, Transition } from "@mantine/core";
import { IconAlertCircle, IconAlertTriangle } from "@tabler/icons-react";

interface Props {
  visible: boolean;
  summary: string;
  detail?: string | null;
  saving?: boolean;
  onSave: () => void;
  onDiscard: () => void;
  /**
   * Renders the bar in the document flow as a banner that sticks to the top
   * of the scroll container, instead of floating over the viewport via Affix.
   * Use this on pages where the staged content fills the screen (e.g. detail
   * pages with tall panels) so the bar stays anchored near the controls the
   * admin is editing rather than drifting far away on tall/wide viewports.
   */
  inline?: boolean;
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
  inline,
  warning,
}: Props) {
  const isWarning = warning != null;
  const saveDisabled = isWarning && !warning.acknowledged;

  const bar = (styles?: CSSProperties) => (
    <Paper
      withBorder
      shadow="md"
      radius="md"
      p="sm"
      pos={inline ? "sticky" : undefined}
      top={inline ? 76 : undefined}
      style={{
        ...styles,
        maxWidth: inline ? undefined : 1100,
        marginInline: inline ? undefined : "auto",
        borderColor: isWarning ? "var(--mantine-color-yellow-6)" : undefined,
      }}
    >
      <Stack gap="sm">
        <Group justify="space-between" wrap="nowrap" gap="md" align="flex-start">
          <Group gap="sm" wrap="nowrap" align="flex-start">
            {isWarning ? (
              <IconAlertTriangle size={20} stroke={1.5} color="var(--mantine-color-yellow-6)" />
            ) : (
              <IconAlertCircle size={20} stroke={1.5} color="var(--mantine-color-orange-6)" />
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
  );

  if (inline) {
    return visible ? bar() : null;
  }

  return (
    <Affix position={{ bottom: 20, left: 20, right: 20 }} zIndex={300}>
      <Transition mounted={visible} transition="slide-up" duration={200}>
        {(styles) => bar(styles)}
      </Transition>
    </Affix>
  );
}
