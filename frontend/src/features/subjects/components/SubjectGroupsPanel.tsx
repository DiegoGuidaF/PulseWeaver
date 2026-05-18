import { Checkbox, Divider, Stack, Switch, Text } from "@mantine/core";
import type { SubjectGroupDetail } from "@/lib/api";
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
      <div>
        <Switch
          label={<Text size="sm" fw={600}>Bypass host check</Text>}
          checked={draft.bypassHostCheck}
          onChange={(e) =>
            dispatch({ type: "setBypass", value: e.currentTarget.checked })
          }
          disabled={disabled}
        />
        <Text size="xs" c="dimmed" mt={4} ml={46}>
          Grants access to ALL hosts, including those not yet in the catalog
        </Text>
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
            <Text size="sm" c="dimmed" p="sm">
              No host groups configured.
            </Text>
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
                  {group.color && (
                    <div
                      style={{
                        width: 8,
                        height: 8,
                        borderRadius: "50%",
                        background: group.color,
                        flexShrink: 0,
                      }}
                    />
                  )}
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
