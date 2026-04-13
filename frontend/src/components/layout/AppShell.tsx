import { Link, useLocation } from "react-router-dom";
import {
    AppShell as MantineAppShell,
    NavLink,
    Text,
    Stack,
    Divider,
    ActionIcon,
    Burger,
    Group,
    useMantineColorScheme,
    useComputedColorScheme,
} from "@mantine/core";
import { useDisclosure } from "@mantine/hooks";
import { BrandName } from "@/components/BrandName";
import {
    IconBrandGithub,
    IconChartBar,
    IconHistory,
    IconList,
    IconLogout,
    IconMessageCircle,
    IconMoon,
    IconServer,
    IconSettings,
    IconSun,
} from "@tabler/icons-react";
import { notifications } from "@mantine/notifications";
import { useLogout } from "@/features/auth/hooks/useLogout";
import { useAuth } from "@/features/auth/hooks/useAuth";
import { useAutoHeartbeat } from "@/features/devices/hooks/useAutoHeartbeat";
import { toErrorMessage } from "@/lib/api-client";

const navItems = [
    { label: "Dashboard", href: "/dashboard", icon: IconChartBar, adminOnly: true },
    { label: "Devices", href: "/devices", icon: IconServer, adminOnly: false },
    { label: "Access Log", href: "/access-log", icon: IconList, adminOnly: true },
    { label: "Address Log", href: "/address-history", icon: IconHistory, adminOnly: true },
    { label: "Settings", href: "/settings", icon: IconSettings, adminOnly: false },
];

function ColorSchemeToggle() {
    const { setColorScheme } = useMantineColorScheme();
    const computed = useComputedColorScheme("light");

    return (
        <ActionIcon
            variant="subtle"
            size="md"
            aria-label="Toggle color scheme"
            onClick={() => setColorScheme(computed === "dark" ? "light" : "dark")}
        >
            {computed === "dark" ? (
                <IconSun size={18} stroke={1.5} />
            ) : (
                <IconMoon size={18} stroke={1.5} />
            )}
        </ActionIcon>
    );
}

export function AppShell({ children }: { children: React.ReactNode }) {
    const [mobileOpened, { toggle: toggleMobile, close: closeMobile }] = useDisclosure();
    const location = useLocation();
    const logoutMutation = useLogout();
    const { user } = useAuth();
    const { clientIp, activeDeviceId } = useAutoHeartbeat();

    return (
        <MantineAppShell
            header={{ height: 60 }}
            navbar={{
                width: 240,
                breakpoint: "md",
                collapsed: { mobile: !mobileOpened },
            }}
            padding="md"
        >
            <MantineAppShell.Header>
                <Group h="100%" px="md" justify="space-between">
                    <Group gap="sm">
                        <Burger
                            opened={mobileOpened}
                            onClick={toggleMobile}
                            hiddenFrom="md"
                            size="sm"
                            aria-label="Toggle navigation"
                        />
                        <BrandName />
                    </Group>
                    <ColorSchemeToggle />
                </Group>
            </MantineAppShell.Header>

            <MantineAppShell.Navbar p="md">
                {/* Nav items */}
                <MantineAppShell.Section grow>
                    <Stack gap={4}>
                        {navItems.filter((item) => !item.adminOnly || user?.role === "admin").map((item) => (
                            <NavLink
                                key={item.href}
                                component={Link}
                                to={item.href}
                                label={item.label}
                                leftSection={<item.icon size={18} stroke={1.5} />}
                                active={location.pathname.startsWith(item.href)}
                                onClick={closeMobile}
                            />
                        ))}
                    </Stack>
                </MantineAppShell.Section>

                {/* Footer: user info, logout, color scheme */}
                <MantineAppShell.Section>
                    <Divider mb="sm" />
                    {user && (
                        <Text size="sm" c="dimmed" mb="xs" px="sm">
                            {user.display_name || user.username}
                        </Text>
                    )}
                    {activeDeviceId && clientIp && (
                        <Group gap="xs" mb="xs" px="sm">
                            <span
                                style={{
                                    display: "inline-block",
                                    width: 8,
                                    height: 8,
                                    borderRadius: "50%",
                                    background: "var(--mantine-color-green-6)",
                                    flexShrink: 0,
                                }}
                            />
                            <Text size="xs" c="dimmed" ff="monospace">
                                {clientIp}
                            </Text>
                        </Group>
                    )}
                    <NavLink
                        component="a"
                        href="https://github.com/DiegoGuidaF/pulseweaver"
                        target="_blank"
                        rel="noopener noreferrer"
                        label="GitHub"
                        leftSection={<IconBrandGithub size={18} stroke={1.5} />}
                        c="dimmed"
                    />
                    <NavLink
                        component="a"
                        href="https://github.com/DiegoGuidaF/pulseweaver/issues"
                        target="_blank"
                        rel="noopener noreferrer"
                        label="Feedback"
                        leftSection={<IconMessageCircle size={18} stroke={1.5} />}
                        c="dimmed"
                    />
                    <Divider my="xs" />
                    <NavLink
                        label={logoutMutation.isPending ? "Logging out…" : "Logout"}
                        leftSection={<IconLogout size={18} stroke={1.5} />}
                        onClick={() => logoutMutation.mutate(
                            {},
                            {
                                onError: (err) =>
                                    notifications.show({ color: "red", title: "Logout failed", message: toErrorMessage(err) }),
                            },
                        )}
                        disabled={logoutMutation.isPending}
                        color="red"
                    />
                </MantineAppShell.Section>
            </MantineAppShell.Navbar>

            <MantineAppShell.Main>{children}</MantineAppShell.Main>
        </MantineAppShell>
    );
}
