import { describe, expect, it, vi } from "vitest";
import { screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderWithProviders } from "@/test/utils";
import {
    createMockPolicyUserEntry,
    createMockPolicyUserIp,
    createMockPolicyUserMapAudit,
} from "@/test/mocks/data";
import { PolicyUserTable } from "./PolicyUserTable";
import type { PolicyUserEntry, PolicyUserMapAudit } from "@/lib/api";

function renderTable(
    users: PolicyUserEntry[],
    overrides?: Partial<PolicyUserMapAudit>,
    onSelectIp = vi.fn(),
    onSelectUser = vi.fn(),
) {
    const data: PolicyUserMapAudit = createMockPolicyUserMapAudit({
        users,
        total_host_count: 5,
        ...overrides,
    });
    return renderWithProviders(
        <PolicyUserTable
            data={data}
            totalHosts={data.total_host_count}
            onSelectIp={onSelectIp}
            onSelectUser={onSelectUser}
        />,
    );
}

// live IPs + host grants → "live_with_access"
const LIVE_WITH_ACCESS = createMockPolicyUserEntry({
    user_id: 1,
    display_name: "alice",
    bypass_allowlist: false,
    allowed_host_count: 2,
    ips: [createMockPolicyUserIp({ ip: "192.168.1.10" })],
});

// bypass
const BYPASS = createMockPolicyUserEntry({
    user_id: 2,
    display_name: "bob",
    bypass_allowlist: true,
    ips: [],
    ip_count: 0,
    device_count: 0,
    allowed_host_count: 0,
    user_allowed_hosts: [],
});

// no live IPs, no grants → "no_access"
const NO_ACCESS = createMockPolicyUserEntry({
    user_id: 3,
    display_name: "carol",
    bypass_allowlist: false,
    ips: [],
    ip_count: 0,
    device_count: 0,
    allowed_host_count: 0,
    user_allowed_hosts: [],
});

// live IPs, but zero host grants → "live_no_host_access" (the key bug case)
const LIVE_NO_HOST_ACCESS = createMockPolicyUserEntry({
    user_id: 4,
    display_name: "dan",
    bypass_allowlist: false,
    allowed_host_count: 0,
    user_allowed_hosts: [],
    ips: [createMockPolicyUserIp({ ip: "10.0.0.5" })],
});

describe("PolicyUserTable", () => {
    it("renders Live + Has access badges for a user with live IPs and host grants", () => {
        renderTable([LIVE_WITH_ACCESS]);

        expect(screen.getByText("Live")).toBeInTheDocument();
        expect(screen.getByText("Has access")).toBeInTheDocument();
    });

    it("renders Bypass badge for bypass users", () => {
        renderTable([BYPASS]);

        expect(screen.getByText("Bypass")).toBeInTheDocument();
    });

    it("renders Offline + No host access badges for users with no live IPs and no grants", () => {
        renderTable([NO_ACCESS]);

        expect(screen.getByText("Offline")).toBeInTheDocument();
        expect(screen.getByText("No host access")).toBeInTheDocument();
    });

    it("renders Live + No host access badges for a revoked user who still has a live IP", () => {
        // Critical case: device is active but allowlist is empty (e.g. after revoke).
        // This must NOT read as "access granted" — it shows live device but no host access.
        renderTable([LIVE_NO_HOST_ACCESS]);

        expect(screen.getByText("Live")).toBeInTheDocument();
        expect(screen.getByText("No host access")).toBeInTheDocument();
    });

    it("search by username filters rows", async () => {
        const user = userEvent.setup();
        renderTable([LIVE_WITH_ACCESS, BYPASS, NO_ACCESS]);

        await user.type(
            screen.getByPlaceholderText(/search by ip, user, or device/i),
            "alice",
        );

        expect(screen.getByText("alice")).toBeInTheDocument();
        expect(screen.queryByText("bob")).not.toBeInTheDocument();
        expect(screen.queryByText("carol")).not.toBeInTheDocument();
        expect(screen.getByText(/1 of 3/)).toBeInTheDocument();
    });

    it("search by IP filters rows", async () => {
        const user = userEvent.setup();
        renderTable([LIVE_WITH_ACCESS, BYPASS, NO_ACCESS]);

        await user.type(
            screen.getByPlaceholderText(/search by ip, user, or device/i),
            "192.168.1.10",
        );

        expect(screen.getByText("alice")).toBeInTheDocument();
        expect(screen.queryByText("bob")).not.toBeInTheDocument();
    });

    it("search by device name filters rows", async () => {
        const user = userEvent.setup();
        const withDevice = createMockPolicyUserEntry({
            user_id: 1,
            display_name: "alice",
            ips: [
                createMockPolicyUserIp({
                    addresses: [
                        {
                            address_id: 1,
                            device_id: 1,
                            device_name: "AlicePhone",
                            updated_at: "2026-01-01T10:00:00Z",
                        },
                    ],
                }),
            ],
        });
        renderTable([withDevice, BYPASS]);

        await user.type(
            screen.getByPlaceholderText(/search by ip, user, or device/i),
            "AlicePhone",
        );

        expect(screen.getByText("alice")).toBeInTheDocument();
        expect(screen.queryByText("bob")).not.toBeInTheDocument();
    });

    it("status filter — bypass shows only bypass users", async () => {
        const user = userEvent.setup();
        renderTable([LIVE_WITH_ACCESS, BYPASS, NO_ACCESS]);

        await user.click(screen.getByRole("radio", { name: /^bypass/i }));

        expect(screen.queryByText("alice")).not.toBeInTheDocument();
        expect(screen.getByText("bob")).toBeInTheDocument();
        expect(screen.queryByText("carol")).not.toBeInTheDocument();
    });

    it("status filter — no access shows only users with no live IPs and no grants", async () => {
        const user = userEvent.setup();
        renderTable([LIVE_WITH_ACCESS, BYPASS, NO_ACCESS, LIVE_NO_HOST_ACCESS]);

        await user.click(screen.getByRole("radio", { name: /^no access/i }));

        expect(screen.queryByText("alice")).not.toBeInTheDocument();
        expect(screen.queryByText("bob")).not.toBeInTheDocument();
        expect(screen.getByText("carol")).toBeInTheDocument();
        expect(screen.queryByText("dan")).not.toBeInTheDocument();
    });

    it("status filter — live no access shows only revoked/empty-allowlist users with live IPs", async () => {
        const user = userEvent.setup();
        renderTable([LIVE_WITH_ACCESS, BYPASS, NO_ACCESS, LIVE_NO_HOST_ACCESS]);

        await user.click(screen.getByRole("radio", { name: /live, no access/i }));

        expect(screen.queryByText("alice")).not.toBeInTheDocument();
        expect(screen.queryByText("bob")).not.toBeInTheDocument();
        expect(screen.queryByText("carol")).not.toBeInTheDocument();
        expect(screen.getByText("dan")).toBeInTheDocument();
    });

    it("shared IPs checkbox filters to shared-only users", async () => {
        const user = userEvent.setup();
        const sharedUser = createMockPolicyUserEntry({
            user_id: 10,
            display_name: "dave",
            on_shared_ip: true,
        });
        renderTable([LIVE_WITH_ACCESS, sharedUser]);

        await user.click(screen.getByRole("checkbox", { name: /shared ips only/i }));

        expect(screen.queryByText("alice")).not.toBeInTheDocument();
        expect(screen.getByText("dave")).toBeInTheDocument();
    });

    it("shows filter empty state when no users match", async () => {
        const user = userEvent.setup();
        renderTable([LIVE_WITH_ACCESS]);

        await user.type(
            screen.getByPlaceholderText(/search by ip, user, or device/i),
            "zzzzz",
        );

        expect(
            screen.getByText("No users match the current filters."),
        ).toBeInTheDocument();
    });

    it("shows empty state when users array is empty", () => {
        renderTable([]);

        expect(screen.getByText("No users in the policy cache.")).toBeInTheDocument();
    });

    it("IP badge click calls onSelectIp but not onSelectUser (stopPropagation)", async () => {
        const user = userEvent.setup();
        const onSelectIp = vi.fn();
        const onSelectUser = vi.fn();
        renderTable([LIVE_WITH_ACCESS], {}, onSelectIp, onSelectUser);

        const ipBadge = screen.getByText(/192\.168\.1\.10/);
        await user.click(ipBadge);

        expect(onSelectIp).toHaveBeenCalledWith("192.168.1.10");
        expect(onSelectUser).not.toHaveBeenCalled();
    });

    it("row click calls onSelectUser with the correct user", async () => {
        const user = userEvent.setup();
        const onSelectUser = vi.fn();
        renderTable([LIVE_WITH_ACCESS], {}, vi.fn(), onSelectUser);

        // Click the row via the user name cell to avoid triggering the IP badge
        const row = screen.getByText("alice").closest("tr")!;
        const nameCell = within(row).getByText("alice");
        await user.click(nameCell);

        expect(onSelectUser).toHaveBeenCalledWith(
            expect.objectContaining({ display_name: "alice" }),
        );
    });

    it("renders Admin badge for admin users", () => {
        const admin = createMockPolicyUserEntry({
            user_id: 5,
            display_name: "superadmin",
            is_admin: true,
        });
        renderTable([admin]);

        expect(screen.getByText("Admin")).toBeInTheDocument();
    });

    it("renders Shared IP badge for users on a shared IP", () => {
        const shared = createMockPolicyUserEntry({
            user_id: 6,
            display_name: "shared-user",
            on_shared_ip: true,
        });
        renderTable([shared]);

        expect(screen.getByText("Shared IP")).toBeInTheDocument();
    });

    it("overflow badge appears when a user has more than 3 IPs", () => {
        const manyIps = createMockPolicyUserEntry({
            user_id: 7,
            display_name: "multiip",
            ips: [
                createMockPolicyUserIp({ ip: "10.0.0.1" }),
                createMockPolicyUserIp({ ip: "10.0.0.2" }),
                createMockPolicyUserIp({ ip: "10.0.0.3" }),
                createMockPolicyUserIp({ ip: "10.0.0.4" }),
            ],
        });
        renderTable([manyIps]);

        // First three are visible as badges; the 4th is collapsed into "+1"
        expect(screen.getByText("+1")).toBeInTheDocument();
    });
});
