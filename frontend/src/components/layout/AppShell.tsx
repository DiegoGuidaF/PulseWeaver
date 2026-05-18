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
    Tooltip,
    ScrollArea,
    Box,
    useMantineColorScheme,
    useComputedColorScheme,
} from "@mantine/core";
import { useDisclosure, useLocalStorage, useMediaQuery } from "@mantine/hooks";
import { BrandName } from "@/components/BrandName";
import {
    IconChartBar,
    IconChevronLeft,
    IconChevronRight,
    IconDatabaseSearch,
    IconHelp,
    IconHistory,
    IconList,
    IconLogout,
    IconMoon,
    IconNetwork,
    IconQrcode,
    IconServer,
    IconSettings,
    IconShield,
    IconSun,
    IconUsers,
} from "@tabler/icons-react";
import { notifications } from "@mantine/notifications";
import { useLogout } from "@/features/auth/hooks/useLogout";
import { useAuth } from "@/features/auth/hooks/useAuth";
import { useAutoHeartbeat } from "@/features/devices/hooks/useAutoHeartbeat";
import { toErrorMessage } from "@/lib/api-client";
import classes from "./AppShell.module.css";

type NavItem = {
    label: string;
    href: string;
    icon: React.ComponentType<{ size?: number; stroke?: number }>;
};

type NavGroup = {
    label: string | null;
    items: NavItem[];
};

const navGroups: NavGroup[] = [
    {
        label: null,
        items: [{ label: "Dashboard", href: "/dashboard", icon: IconChartBar }],
    },
    {
        label: "Devices",
        items: [
            { label: "Devices", href: "/devices", icon: IconServer },
            { label: "Provisioning", href: "/device-provisioning", icon: IconQrcode },
        ],
    },
    {
        label: "Access",
        items: [
            { label: "Hosts", href: "/hosts", icon: IconShield },
            { label: "Users", href: "/users", icon: IconUsers },
            { label: "Network Policies", href: "/network-policies", icon: IconNetwork },
        ],
    },
    {
        label: "Auditing",
        items: [
            { label: "Access Logs", href: "/access-log", icon: IconList },
            { label: "IP Address Logs", href: "/address-history", icon: IconHistory },
            { label: "Access Policy Cache", href: "/policy-audit", icon: IconDatabaseSearch },
        ],
    },
    {
        label: null,
        items: [{ label: "Settings", href: "/settings", icon: IconSettings }],
    },
];

const collapsedItemStyles = {
    root: { justifyContent: "center" as const, padding: "8px" },
    body: { display: "none" as const },
    section: { margin: 0 },
};

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
    const [navCollapsed, setNavCollapsed] = useLocalStorage({
        key: "pw-nav-collapsed",
        defaultValue: false,
        getInitialValueInEffect: false,
    });
    // Assume desktop on first render to avoid content/width mismatch for users with stored collapsed state
    const isMd = useMediaQuery("(min-width: 62em)", true, { getInitialValueInEffect: false });
    const isCollapsed = navCollapsed && isMd;

    const location = useLocation();
    const logoutMutation = useLogout();
    const { user } = useAuth();
    const { clientIp, activeDeviceId } = useAutoHeartbeat();

    return (
        <MantineAppShell
            header={{ height: 60 }}
            navbar={{
                width: { base: 240, md: isCollapsed ? 60 : 240 },
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
                    <Group gap="xs">
                        <ColorSchemeToggle />
                        <Tooltip label="Help & docs" position="bottom">
                            <ActionIcon
                                component="a"
                                href="https://github.com/DiegoGuidaF/pulseweaver"
                                target="_blank"
                                rel="noopener noreferrer"
                                variant="subtle"
                                size="md"
                                aria-label="Help and docs"
                            >
                                <IconHelp size={18} stroke={1.5} />
                            </ActionIcon>
                        </Tooltip>
                    </Group>
                </Group>
            </MantineAppShell.Header>

            <MantineAppShell.Navbar className={classes.navbar} p={0}>
                {/* Scrollable nav groups */}
                <MantineAppShell.Section grow component={ScrollArea}>
                    <Stack gap={2} px="xs" pt="xs" pb="xs">
                        {navGroups.map((group, groupIdx) => (
                            <Box key={groupIdx}>
                                {groupIdx > 0 && !isCollapsed && (
                                    group.label
                                        ? <Text className={classes.sectionLabel}>{group.label}</Text>
                                        : <Divider my={4} />
                                )}
                                {group.items.map(item => {
                                    const isActive = location.pathname.startsWith(item.href);
                                    if (isCollapsed) {
                                        return (
                                            <Tooltip key={item.href} label={item.label} position="right" withArrow>
                                                <NavLink
                                                    component={Link}
                                                    to={item.href}
                                                    leftSection={<item.icon size={18} stroke={1.5} />}
                                                    active={isActive}
                                                    onClick={closeMobile}
                                                    styles={collapsedItemStyles}
                                                    aria-label={item.label}
                                                />
                                            </Tooltip>
                                        );
                                    }
                                    return (
                                        <NavLink
                                            key={item.href}
                                            component={Link}
                                            to={item.href}
                                            label={item.label}
                                            leftSection={<item.icon size={18} stroke={1.5} />}
                                            active={isActive}
                                            onClick={closeMobile}
                                            className={classes.navItem}
                                        />
                                    );
                                })}
                            </Box>
                        ))}
                    </Stack>
                </MantineAppShell.Section>

                {/* Footer: collapse toggle, user info, logout */}
                <MantineAppShell.Section>
                    <Box px="xs" pb="xs">
                        {/* Collapse toggle — desktop only */}
                        <Group justify={isCollapsed ? "center" : "flex-end"} mb="xs" visibleFrom="md">
                            <Tooltip
                                label={navCollapsed ? "Expand sidebar" : "Collapse sidebar"}
                                position="right"
                                withArrow
                            >
                                <ActionIcon
                                    variant="subtle"
                                    size="sm"
                                    onClick={() => setNavCollapsed(!navCollapsed)}
                                    aria-label={navCollapsed ? "Expand sidebar" : "Collapse sidebar"}
                                >
                                    {navCollapsed
                                        ? <IconChevronRight size={14} stroke={1.5} />
                                        : <IconChevronLeft size={14} stroke={1.5} />
                                    }
                                </ActionIcon>
                            </Tooltip>
                        </Group>

                        <Divider mb="sm" />

                        {!isCollapsed && user && (
                            <Text size="sm" c="dimmed" mb="xs" px="sm">
                                {user.display_name || user.username}
                            </Text>
                        )}

                        {activeDeviceId && clientIp && (
                            isCollapsed ? (
                                <Tooltip label={clientIp} position="right" withArrow>
                                    <Group justify="center" mb="xs" style={{ cursor: "default" }}>
                                        <span className={classes.heartbeatDot} />
                                    </Group>
                                </Tooltip>
                            ) : (
                                <Group gap="xs" mb="xs" px="sm">
                                    <span className={classes.heartbeatDot} />
                                    <Text size="xs" c="dimmed" ff="monospace">
                                        {clientIp}
                                    </Text>
                                </Group>
                            )
                        )}

                        <Divider my="xs" />

                        {isCollapsed ? (
                            <Tooltip
                                label={logoutMutation.isPending ? "Logging out…" : "Logout"}
                                position="right"
                                withArrow
                            >
                                <NavLink
                                    leftSection={<IconLogout size={18} stroke={1.5} />}
                                    onClick={() => logoutMutation.mutate({}, {
                                        onError: (err) =>
                                            notifications.show({ color: "red", title: "Logout failed", message: toErrorMessage(err) }),
                                    })}
                                    disabled={logoutMutation.isPending}
                                    color="red"
                                    styles={collapsedItemStyles}
                                    aria-label={logoutMutation.isPending ? "Logging out…" : "Logout"}
                                />
                            </Tooltip>
                        ) : (
                            <NavLink
                                label={logoutMutation.isPending ? "Logging out…" : "Logout"}
                                leftSection={<IconLogout size={18} stroke={1.5} />}
                                onClick={() => logoutMutation.mutate({}, {
                                    onError: (err) =>
                                        notifications.show({ color: "red", title: "Logout failed", message: toErrorMessage(err) }),
                                })}
                                disabled={logoutMutation.isPending}
                                color="red"
                            />
                        )}
                    </Box>
                </MantineAppShell.Section>
            </MantineAppShell.Navbar>

            <MantineAppShell.Main>{children}</MantineAppShell.Main>
        </MantineAppShell>
    );
}
