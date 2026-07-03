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
} from "@mantine/core";
import { useDisclosure, useLocalStorage, useMediaQuery } from "@mantine/hooks";
import { BrandName } from "@/components/BrandName";
import {
    IconChartBar,
    IconChevronLeft,
    IconChevronRight,
    IconDatabaseSearch,
    IconFolder,
    IconHistory,
    IconList,
    IconNetwork,
    IconServer,
    IconShield,
    IconUsers,
} from "@tabler/icons-react";
import { ROUTES } from "@/lib/routes";
import { UserMenu } from "./UserMenu";
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
        items: [{ label: "Dashboard", href: ROUTES.dashboard, icon: IconChartBar }],
    },
    {
        label: "Devices",
        items: [
            { label: "Devices", href: ROUTES.devices, icon: IconServer },
        ],
    },
    {
        label: "Access",
        items: [
            { label: "Hosts", href: ROUTES.accessHosts, icon: IconShield },
            { label: "Host Groups", href: ROUTES.accessHostGroups, icon: IconFolder },
            { label: "Users", href: ROUTES.accessUsers, icon: IconUsers },
            { label: "Network Policies", href: ROUTES.accessNetworkPolicies, icon: IconNetwork },
        ],
    },
    {
        label: "Auditing",
        items: [
            { label: "Access Logs", href: ROUTES.accessLog, icon: IconList },
            { label: "IP Address Logs", href: ROUTES.addressHistory, icon: IconHistory },
            { label: "Access Verification", href: ROUTES.policyAudit, icon: IconDatabaseSearch },
        ],
    },
];

const collapsedItemStyles = {
    root: { justifyContent: "center" as const, padding: "8px" },
    body: { display: "none" as const },
    section: { margin: 0 },
};

export function AppShell({ children }: { children: React.ReactNode }) {
    const [mobileOpened, { toggle: toggleMobile, close: closeMobile }] = useDisclosure();
    const [navCollapsed, setNavCollapsed] = useLocalStorage({
        key: "pw-nav-collapsed",
        defaultValue: false,
        getInitialValueInEffect: false,
    });
    const [navWidth, setNavWidth] = useLocalStorage({
        key: "pw-nav-width",
        defaultValue: 240,
        getInitialValueInEffect: false,
    });
    // Assume desktop on first render to avoid content/width mismatch for users with stored collapsed state
    const isMd = useMediaQuery("(min-width: 62em)", true, { getInitialValueInEffect: false });
    const isCollapsed = navCollapsed && isMd;

    const handleResizeMouseDown = (e: React.MouseEvent) => {
        if (!isMd) return;
        e.preventDefault();
        const startX = e.clientX;
        const startWidth = navWidth;
        let wasDragging = false;

        const onMove = (ev: MouseEvent) => {
            const delta = ev.clientX - startX;
            if (!wasDragging && Math.abs(delta) > 5) wasDragging = true;
            if (wasDragging) {
                setNavWidth(Math.max(160, Math.min(400, startWidth + delta)));
                if (navCollapsed) setNavCollapsed(false);
            }
        };

        const onUp = () => {
            if (!wasDragging) setNavCollapsed(!navCollapsed);
            document.removeEventListener("mousemove", onMove);
            document.removeEventListener("mouseup", onUp);
        };

        document.addEventListener("mousemove", onMove);
        document.addEventListener("mouseup", onUp);
    };

    const location = useLocation();

    return (
        <MantineAppShell
            header={{ height: 60 }}
            navbar={{
                width: { base: 240, md: isCollapsed ? 60 : navWidth },
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
                    <UserMenu />
                </Group>
            </MantineAppShell.Header>

            <MantineAppShell.Navbar className={classes.navbar} p={0}>
                <Box
                    className={classes.resizeHandle}
                    visibleFrom="md"
                    onMouseDown={handleResizeMouseDown}
                />
                {/* Scrollable nav groups */}
                <MantineAppShell.Section grow component={ScrollArea}>
                    <Stack gap={2} px="xs" pt="xs" pb="xs">
                        {navGroups.map((group, groupIdx) => (
                            <Box key={groupIdx}>
                                {groupIdx > 0 && (
                                    isCollapsed
                                        ? <Divider my={6} mx="sm" />
                                        : group.label
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

                {/* Footer: desktop sidebar collapse toggle (account + logout live in the top-bar user menu) */}
                <MantineAppShell.Section visibleFrom="md">
                    <Box px="xs" pb="xs">
                        <Group justify={isCollapsed ? "center" : "flex-end"}>
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
                    </Box>
                </MantineAppShell.Section>
            </MantineAppShell.Navbar>

            <MantineAppShell.Main>{children}</MantineAppShell.Main>
        </MantineAppShell>
    );
}
