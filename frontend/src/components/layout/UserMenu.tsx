import { Link } from "react-router-dom";
import {
  Avatar,
  Button,
  Divider,
  Group,
  Popover,
  SegmentedControl,
  Stack,
  Text,
  UnstyledButton,
  useComputedColorScheme,
  useMantineColorScheme,
} from "@mantine/core";
import { useDisclosure } from "@mantine/hooks";
import { notifications } from "@mantine/notifications";
import {
  IconChevronDown,
  IconHelp,
  IconLogout,
  IconUserCog,
} from "@tabler/icons-react";
import { useAuth } from "@/features/auth/hooks/useAuth";
import { useLogout } from "@/features/auth/hooks/useLogout";
import { toErrorMessage } from "@/lib/api-client";
import { ROUTES } from "@/lib/routes";
import { DateTimePrefsPanel } from "./DateTimePrefsPanel";

const HELP_URL = "https://github.com/DiegoGuidaF/pulseweaver";

function initials(name: string): string {
  const parts = name.trim().split(/\s+/).filter(Boolean);
  if (parts.length === 0) return "?";
  if (parts.length === 1) return parts[0].slice(0, 2).toUpperCase();
  return (parts[0][0] + parts[parts.length - 1][0]).toUpperCase();
}

function ThemeControl() {
  const { setColorScheme } = useMantineColorScheme();
  const computed = useComputedColorScheme("light");

  return (
    <SegmentedControl
      fullWidth
      size="xs"
      value={computed}
      onChange={(val) => setColorScheme(val as "light" | "dark")}
      data={[
        { label: "Light", value: "light" },
        { label: "Dark", value: "dark" },
      ]}
    />
  );
}

export function UserMenu() {
  const [opened, { toggle, close }] = useDisclosure(false);
  const { user } = useAuth();
  const logoutMutation = useLogout();

  const displayName = user?.display_name || user?.username || "Account";

  return (
    <Popover
      opened={opened}
      onChange={(o) => (o ? toggle() : close())}
      position="bottom-end"
      width={260}
      shadow="md"
      withArrow
    >
      <Popover.Target>
        <UnstyledButton
          onClick={toggle}
          aria-label="Open account menu"
          style={{ padding: "4px 8px", borderRadius: "var(--mantine-radius-sm)" }}
        >
          <Group gap="xs" wrap="nowrap">
            <Avatar size={28} radius="xl" color="indigo">
              {initials(displayName)}
            </Avatar>
            <Text size="sm" fw={500} visibleFrom="sm" maw={140} truncate="end">
              {displayName}
            </Text>
            <IconChevronDown size={16} stroke={1.5} />
          </Group>
        </UnstyledButton>
      </Popover.Target>

      <Popover.Dropdown p="xs">
        <Stack gap="xs">
          <div>
            <Text size="sm" fw={600} truncate="end">{displayName}</Text>
            {user?.username && (
              <Text size="xs" c="dimmed" truncate="end">@{user.username}</Text>
            )}
          </div>

          <Divider />

          <Button
            component={Link}
            to={ROUTES.account}
            onClick={close}
            variant="subtle"
            color="gray"
            fullWidth
            justify="flex-start"
            leftSection={<IconUserCog size={18} stroke={1.5} />}
          >
            Account settings
          </Button>

          <Divider />

          <div>
            <Text size="xs" fw={500} mb={4}>Theme</Text>
            <ThemeControl />
          </div>

          <Divider />

          <DateTimePrefsPanel />

          <Divider />

          <Button
            component="a"
            href={HELP_URL}
            target="_blank"
            rel="noopener noreferrer"
            onClick={close}
            variant="subtle"
            color="gray"
            fullWidth
            justify="flex-start"
            leftSection={<IconHelp size={18} stroke={1.5} />}
          >
            Help &amp; docs
          </Button>

          <Button
            variant="subtle"
            color="red"
            fullWidth
            justify="flex-start"
            leftSection={<IconLogout size={18} stroke={1.5} />}
            loading={logoutMutation.isPending}
            onClick={() =>
              logoutMutation.mutate(
                {},
                {
                  onError: (err) =>
                    notifications.show({
                      color: "red",
                      title: "Logout failed",
                      message: toErrorMessage(err),
                    }),
                },
              )
            }
          >
            Log out
          </Button>
        </Stack>
      </Popover.Dropdown>
    </Popover>
  );
}
