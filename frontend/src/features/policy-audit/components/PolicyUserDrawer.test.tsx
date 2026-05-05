import { describe, expect, it, vi } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderWithProviders } from "@/test/utils";
import {
    createMockPolicyUserEntry,
    createMockPolicyUserIp,
    createMockPolicyUserAddress,
} from "@/test/mocks/data";
import { PolicyUserDrawer } from "./PolicyUserDrawer";
import type { PolicyUserEntry } from "@/lib/api";
import { TEST_TIMEOUTS } from "@/test/constants";

function renderDrawer(
    user: PolicyUserEntry | null,
    totalHosts = 5,
    onClose = vi.fn(),
    onSelectIp = vi.fn(),
) {
    return renderWithProviders(
        <PolicyUserDrawer
            user={user}
            totalHosts={totalHosts}
            onClose={onClose}
            onSelectIp={onSelectIp}
        />,
    );
}

const ALLOWLISTED_USER = createMockPolicyUserEntry({
    user_id: 1,
    user_name: "alice",
    bypass_allowlist: false,
    on_shared_ip: false,
    is_admin: false,
    user_allowed_hosts: ["app.home.lan", "media.home.lan"],
    ips: [
        createMockPolicyUserIp({
            ip: "192.168.1.10",
            effective_hosts: ["app.home.lan", "media.home.lan"],
            addresses: [
                createMockPolicyUserAddress({ device_name: "AliceLaptop" }),
            ],
        }),
    ],
    device_count: 1,
    allowed_host_count: 2,
});

const BYPASS_USER = createMockPolicyUserEntry({
    user_id: 2,
    user_name: "bob",
    bypass_allowlist: true,
    on_shared_ip: false,
    ips: [],
    ip_count: 0,
    device_count: 0,
    allowed_host_count: 0,
    user_allowed_hosts: [],
});

const NO_ACCESS_USER = createMockPolicyUserEntry({
    user_id: 3,
    user_name: "carol",
    bypass_allowlist: false,
    ips: [],
    ip_count: 0,
    device_count: 0,
    allowed_host_count: 0,
    user_allowed_hosts: [],
});

describe("PolicyUserDrawer", () => {
    it("is not visible when user is null", () => {
        renderDrawer(null);

        expect(screen.queryByText("User · Policy")).not.toBeInTheDocument();
    });

    it("shows username and Allowlisted badge for an allowlisted user", async () => {
        renderDrawer(ALLOWLISTED_USER);

        await waitFor(
            () => expect(screen.getByText("alice")).toBeInTheDocument(),
            { timeout: TEST_TIMEOUTS.SHORT },
        );

        expect(screen.getByText("Allowlisted")).toBeInTheDocument();
    });

    it("shows Bypass badge for a bypass user", async () => {
        renderDrawer(BYPASS_USER);

        await waitFor(
            () => expect(screen.getByText("bob")).toBeInTheDocument(),
            { timeout: TEST_TIMEOUTS.SHORT },
        );

        expect(screen.getByText("Bypass")).toBeInTheDocument();
    });

    it("shows No access badge for a no-access user", async () => {
        renderDrawer(NO_ACCESS_USER);

        await waitFor(
            () => expect(screen.getByText("carol")).toBeInTheDocument(),
            { timeout: TEST_TIMEOUTS.SHORT },
        );

        expect(screen.getByText("No access")).toBeInTheDocument();
    });

    it("shows Admin badge for admin users", async () => {
        const admin = createMockPolicyUserEntry({
            ...ALLOWLISTED_USER,
            user_name: "dana",
            is_admin: true,
        });
        renderDrawer(admin);

        await waitFor(
            () => expect(screen.getByText("dana")).toBeInTheDocument(),
            { timeout: TEST_TIMEOUTS.SHORT },
        );

        expect(screen.getByText("Admin")).toBeInTheDocument();
    });

    it("shows Shared IP badge and alert for users on a shared IP", async () => {
        const shared = createMockPolicyUserEntry({
            ...ALLOWLISTED_USER,
            user_name: "eve",
            on_shared_ip: true,
        });
        renderDrawer(shared);

        await waitFor(
            () => expect(screen.getByText("eve")).toBeInTheDocument(),
            { timeout: TEST_TIMEOUTS.SHORT },
        );

        expect(screen.getByText("Shared IP")).toBeInTheDocument();
        expect(
            screen.getByText("Shares an IP with another user"),
        ).toBeInTheDocument();
    });

    it("Hosts tab renders allowed host FQDNs for allowlisted user", async () => {
        renderDrawer(ALLOWLISTED_USER);

        await waitFor(
            () => expect(screen.getByText("app.home.lan")).toBeInTheDocument(),
            { timeout: TEST_TIMEOUTS.SHORT },
        );

        expect(screen.getByText("media.home.lan")).toBeInTheDocument();
    });

    it("Hosts tab shows bypass message for bypass user", async () => {
        renderDrawer(BYPASS_USER);

        await waitFor(
            () =>
                expect(
                    screen.getByText(/bypass enabled — all system hosts accessible/i),
                ).toBeInTheDocument(),
            { timeout: TEST_TIMEOUTS.SHORT },
        );
    });

    it("Live IPs tab shows the user's IP card", async () => {
        const user = userEvent.setup();
        renderDrawer(ALLOWLISTED_USER);

        await waitFor(
            () => expect(screen.getByText("alice")).toBeInTheDocument(),
            { timeout: TEST_TIMEOUTS.SHORT },
        );

        await user.click(screen.getByRole("tab", { name: /live ips/i }));

        await waitFor(
            () => expect(screen.getByText("192.168.1.10")).toBeInTheDocument(),
            { timeout: TEST_TIMEOUTS.SHORT },
        );
    });

    it("Live IPs tab shows empty state when user has no IPs", async () => {
        const user = userEvent.setup();
        renderDrawer(NO_ACCESS_USER);

        await waitFor(
            () => expect(screen.getByText("carol")).toBeInTheDocument(),
            { timeout: TEST_TIMEOUTS.SHORT },
        );

        await user.click(screen.getByRole("tab", { name: /live ips/i }));

        await waitFor(
            () =>
                expect(
                    screen.getByText("No live IPs in the cache."),
                ).toBeInTheDocument(),
            { timeout: TEST_TIMEOUTS.SHORT },
        );
    });

    it('"Test from this IP" calls onSelectIp and onClose', async () => {
        const user = userEvent.setup();
        const onSelectIp = vi.fn();
        const onClose = vi.fn();
        renderDrawer(ALLOWLISTED_USER, 5, onClose, onSelectIp);

        await waitFor(
            () => expect(screen.getByText("alice")).toBeInTheDocument(),
            { timeout: TEST_TIMEOUTS.SHORT },
        );

        await user.click(screen.getByRole("tab", { name: /live ips/i }));

        const testButton = await screen.findByRole("button", {
            name: /test from this ip/i,
        });
        await user.click(testButton);

        expect(onSelectIp).toHaveBeenCalledWith("192.168.1.10");
        expect(onClose).toHaveBeenCalled();
    });

    it("Devices tab shows device name", async () => {
        const user = userEvent.setup();
        renderDrawer(ALLOWLISTED_USER);

        await waitFor(
            () => expect(screen.getByText("alice")).toBeInTheDocument(),
            { timeout: TEST_TIMEOUTS.SHORT },
        );

        await user.click(screen.getByRole("tab", { name: /devices/i }));

        // Mantine renders all tab panels in the DOM simultaneously; "AliceLaptop"
        // may appear in the Live IPs panel too (device name under the IP card).
        // Assert presence without requiring a single match.
        await waitFor(
            () =>
                expect(screen.getAllByText("AliceLaptop").length).toBeGreaterThan(0),
            { timeout: TEST_TIMEOUTS.SHORT },
        );

        // "1 live IP" label is unique to the Devices tab's device card
        expect(screen.getByText("1 live IP")).toBeInTheDocument();
    });

    it("Devices tab shows empty state when user has no devices", async () => {
        const user = userEvent.setup();
        renderDrawer(NO_ACCESS_USER);

        await waitFor(
            () => expect(screen.getByText("carol")).toBeInTheDocument(),
            { timeout: TEST_TIMEOUTS.SHORT },
        );

        await user.click(screen.getByRole("tab", { name: /devices/i }));

        await waitFor(
            () =>
                expect(
                    screen.getByText("No devices in the cache."),
                ).toBeInTheDocument(),
            { timeout: TEST_TIMEOUTS.SHORT },
        );
    });

    it("Hosts tab filter buttons narrow host list", async () => {
        const user = userEvent.setup();
        // trimmed_hosts is kept empty so "media.home.lan" only appears in the
        // HostsTab host list — avoiding a duplicate match from the IpCard alert
        // that Mantine always renders in the DOM even for inactive tab panels.
        const partialUser = createMockPolicyUserEntry({
            user_id: 10,
            user_name: "frank",
            bypass_allowlist: false,
            user_allowed_hosts: ["app.home.lan", "media.home.lan"],
            allowed_host_count: 2,
            ips: [
                createMockPolicyUserIp({
                    ip: "10.0.0.1",
                    effective_hosts: ["app.home.lan"],
                    trimmed_hosts: [],
                }),
            ],
        });
        renderDrawer(partialUser, 5);

        await waitFor(
            () => expect(screen.getByText("app.home.lan")).toBeInTheDocument(),
            { timeout: TEST_TIMEOUTS.SHORT },
        );

        // Both hosts visible in "All" (default)
        expect(screen.getByText("media.home.lan")).toBeInTheDocument();

        // "Reachable" appears in both the StatsRow label and the HostsTab filter
        // badge ("Reachable · N"). The badge text includes "·" which is unique.
        const reachableBadge = screen
            .getAllByText(/Reachable/)
            .find((el) => el.textContent?.includes("·"))!;
        await user.click(reachableBadge);

        expect(screen.getByText("app.home.lan")).toBeInTheDocument();
        expect(screen.queryByText("media.home.lan")).not.toBeInTheDocument();
    });
});
