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

const ALLOWLISTED = createMockPolicyUserEntry({
    user_id: 1,
    user_name: "alice",
    bypass_allowlist: false,
    ips: [createMockPolicyUserIp({ ip: "192.168.1.10" })],
});
const BYPASS = createMockPolicyUserEntry({
    user_id: 2,
    user_name: "bob",
    bypass_allowlist: true,
    ips: [],
    ip_count: 0,
    device_count: 0,
    allowed_host_count: 0,
    user_allowed_hosts: [],
});
const NO_ACCESS = createMockPolicyUserEntry({
    user_id: 3,
    user_name: "carol",
    bypass_allowlist: false,
    ips: [],
    ip_count: 0,
    device_count: 0,
    allowed_host_count: 0,
    user_allowed_hosts: [],
});

describe("PolicyUserTable", () => {
    it("renders all three status badges", () => {
        renderTable([ALLOWLISTED, BYPASS, NO_ACCESS]);

        expect(screen.getByText("Allowlisted")).toBeInTheDocument();
        expect(screen.getByText("Bypass")).toBeInTheDocument();
        expect(screen.getByText("No access")).toBeInTheDocument();
    });

    it("search by username filters rows", async () => {
        const user = userEvent.setup();
        renderTable([ALLOWLISTED, BYPASS, NO_ACCESS]);

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
        renderTable([ALLOWLISTED, BYPASS, NO_ACCESS]);

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
            user_name: "alice",
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
        renderTable([ALLOWLISTED, BYPASS, NO_ACCESS]);

        await user.click(screen.getByRole("radio", { name: /^bypass/i }));

        expect(screen.queryByText("alice")).not.toBeInTheDocument();
        expect(screen.getByText("bob")).toBeInTheDocument();
        expect(screen.queryByText("carol")).not.toBeInTheDocument();
    });

    it("status filter — no access shows only no-access users", async () => {
        const user = userEvent.setup();
        renderTable([ALLOWLISTED, BYPASS, NO_ACCESS]);

        await user.click(screen.getByRole("radio", { name: /^no access/i }));

        expect(screen.queryByText("alice")).not.toBeInTheDocument();
        expect(screen.queryByText("bob")).not.toBeInTheDocument();
        expect(screen.getByText("carol")).toBeInTheDocument();
    });

    it("shared IPs checkbox filters to shared-only users", async () => {
        const user = userEvent.setup();
        const sharedUser = createMockPolicyUserEntry({
            user_id: 4,
            user_name: "dave",
            on_shared_ip: true,
        });
        renderTable([ALLOWLISTED, sharedUser]);

        await user.click(screen.getByRole("checkbox", { name: /shared ips only/i }));

        expect(screen.queryByText("alice")).not.toBeInTheDocument();
        expect(screen.getByText("dave")).toBeInTheDocument();
    });

    it("shows filter empty state when no users match", async () => {
        const user = userEvent.setup();
        renderTable([ALLOWLISTED]);

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
        renderTable([ALLOWLISTED], {}, onSelectIp, onSelectUser);

        const ipBadge = screen.getByText(/192\.168\.1\.10/);
        await user.click(ipBadge);

        expect(onSelectIp).toHaveBeenCalledWith("192.168.1.10");
        expect(onSelectUser).not.toHaveBeenCalled();
    });

    it("row click calls onSelectUser with the correct user", async () => {
        const user = userEvent.setup();
        const onSelectUser = vi.fn();
        renderTable([ALLOWLISTED], {}, vi.fn(), onSelectUser);

        // Click the row via the user name cell to avoid triggering the IP badge
        const row = screen.getByText("alice").closest("tr")!;
        const nameCell = within(row).getByText("alice");
        await user.click(nameCell);

        expect(onSelectUser).toHaveBeenCalledWith(
            expect.objectContaining({ user_name: "alice" }),
        );
    });

    it("renders Admin badge for admin users", () => {
        const admin = createMockPolicyUserEntry({
            user_id: 5,
            user_name: "superadmin",
            is_admin: true,
        });
        renderTable([admin]);

        expect(screen.getByText("Admin")).toBeInTheDocument();
    });

    it("renders Shared IP badge for users on a shared IP", () => {
        const shared = createMockPolicyUserEntry({
            user_id: 6,
            user_name: "shared-user",
            on_shared_ip: true,
        });
        renderTable([shared]);

        expect(screen.getByText("Shared IP")).toBeInTheDocument();
    });

    it("overflow badge appears when a user has more than 3 IPs", () => {
        const manyIps = createMockPolicyUserEntry({
            user_id: 7,
            user_name: "multiip",
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
