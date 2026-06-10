import { Button, Checkbox, Divider, Stack, Switch, Text, ThemeIcon } from "@mantine/core";
import { Link } from "react-router-dom";
import { IconAlertTriangle, IconArrowRight } from "@tabler/icons-react";
import type { SubjectGroupDetail } from "@/lib/api";
import { resolveGroupIcon } from "@/features/host-access/hostIconConfig";
import { ROUTES } from "@/lib/routes";
import type { SubjectAccessDraft, SubjectAccessAction } from "../drafts/subjectAccessDraft";

interface Props {
  groups: SubjectGroupDetail[];
  draft: SubjectAccessDraft;
  dispatch: React.Dispatch<SubjectAccessAction>;
  disabled?: boolean;
}

export function SubjectGroupsPanel({ groups, draft, dispatch, disabled }: Props) {
  return (
    <Stack gap="md">
      <div
        style={{
          border: "1px solid var(--mantine-color-yellow-7)",
          borderRadius: "var(--mantine-radius-sm)",
          padding: "12px",
          backgroundColor: "var(--mantine-color-yellow-light)",
        }}
      >
        <Switch
          label={
            <Stack gap={2}>
              <Text size="sm" fw={700} c="yellow.5">
                Bypass host check
              </Text>
              <Text size="xs" c="dimmed">
                Grants access to ALL hosts, including those not yet in the catalog —
                this overrides every group assignment below.
              </Text>
            </Stack>
          }
          color="yellow"
          thumbIcon={
            draft.bypassHostCheck ? (
              <IconAlertTriangle size={12} color="var(--mantine-color-yellow-9)" stroke={3} />
            ) : undefined
          }
          checked={draft.bypassHostCheck}
          onChange={(e) =>
            dispatch({ type: "setBypass", value: e.currentTarget.checked })
          }
          disabled={disabled}
        />
      </div>

      <Divider />

      <div>
        <Text size="sm" fw={600} mb={4}>
          Groups
          {draft.bypassHostCheck && (
            <Text span size="xs" c="dimmed" fw={400} ml="xs">
              · suspended while bypass is active
            </Text>
          )}
        </Text>
        <Text size="xs" c="dimmed" mb="xs">
          Check to assign · uncheck to remove
        </Text>
        <div
          style={{
            opacity: draft.bypassHostCheck ? 0.45 : 1,
            pointerEvents: draft.bypassHostCheck ? "none" : undefined,
            border: "1px solid var(--mantine-color-default-border)",
            borderRadius: "var(--mantine-radius-sm)",
            overflow: "hidden",
          }}
        >
          {groups.length === 0 ? (
            <Stack gap="xs" p="sm" align="flex-start">
              <Text size="sm" c="dimmed">
                No host groups yet — create at least one to start assigning access.
              </Text>
              <Button
                component={Link}
                to={ROUTES.accessHostGroups}
                variant="light"
                size="xs"
                rightSection={<IconArrowRight size={14} />}
              >
                Go to Host Groups
              </Button>
            </Stack>
          ) : (
            groups.map((group, i) => {
              const isAssigned = draft.assignedGroupIds.has(group.id);
              return (
                <div
                  key={group.id}
                  style={{
                    display: "flex",
                    alignItems: "center",
                    gap: 10,
                    padding: "8px 12px",
                    borderBottom:
                      i < groups.length - 1
                        ? "1px solid var(--mantine-color-default-border)"
                        : undefined,
                  }}
                >
                  <Checkbox
                    size="md"
                    checked={isAssigned}
                    disabled={disabled || draft.bypassHostCheck}
                    onChange={(e) =>
                      dispatch({
                        type: "toggleGroup",
                        id: group.id,
                        assigned: e.currentTarget.checked,
                      })
                    }
                  />
                  <ThemeIcon variant="light" color={group.color} size="sm" radius="sm">
                    {resolveGroupIcon(group.icon)({ size: 12 })}
                  </ThemeIcon>
                  <Text size="sm" style={{ flex: 1 }}>
                    {group.name}
                  </Text>
                  <Text size="xs" c="dimmed">
                    {group.hosts.length} {group.hosts.length === 1 ? "host" : "hosts"}
                  </Text>
                </div>
              );
            })
          )}
        </div>
      </div>
    </Stack>
  );
}
