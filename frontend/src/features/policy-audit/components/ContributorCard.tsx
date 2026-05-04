import { Badge, Card, Group, Stack, Text } from "@mantine/core";
import { IconAlertTriangle } from "@tabler/icons-react";
import type { PolicyMapContributor } from "@/lib/api";
import { useDateFormatter } from "@/contexts/useDateTimePrefs";

interface ContributorCardProps {
  contributor: PolicyMapContributor;
}

export function ContributorCard({ contributor }: ContributorCardProps) {
  const formatDateTime = useDateFormatter();
  const trimmedSet = new Set(contributor.trimmed_hosts);

  return (
    <Card withBorder p="sm" radius="sm">
      <Stack gap="xs">
        <Group justify="space-between" align="flex-start">
          <div>
            <Text size="sm" fw={500}>
              {contributor.user_name}
            </Text>
            <Text size="xs" c="dimmed">
              {contributor.device_name} · updated {formatDateTime(contributor.address_updated_at)}
            </Text>
          </div>
          {contributor.user_bypass && (
            <Badge variant="light" color="green" size="sm">
              Bypasses allowlist
            </Badge>
          )}
        </Group>

        {!contributor.user_bypass && (
          <>
            {contributor.user_allowed_hosts.length === 0 ? (
              <Text size="xs" c="dimmed">
                No hosts granted
              </Text>
            ) : (
              <Group gap={4} wrap="wrap">
                {contributor.user_allowed_hosts.map((h) =>
                  trimmedSet.has(h) ? (
                    <Badge
                      key={h}
                      variant="light"
                      color="gray"
                      size="sm"
                      style={{ textDecoration: "line-through", opacity: 0.5 }}
                    >
                      {h}
                    </Badge>
                  ) : (
                    <Badge key={h} variant="light" color="indigo" size="sm">
                      {h}
                    </Badge>
                  ),
                )}
              </Group>
            )}

            {contributor.trimmed_hosts.length > 0 && (
              <Group gap={4}>
                <IconAlertTriangle size={14} color="var(--mantine-color-yellow-6)" />
                <Text size="xs" c="yellow.7">
                  {contributor.trimmed_hosts.length}{" "}
                  {contributor.trimmed_hosts.length === 1 ? "host" : "hosts"} trimmed by
                  intersection
                </Text>
              </Group>
            )}
          </>
        )}
      </Stack>
    </Card>
  );
}
