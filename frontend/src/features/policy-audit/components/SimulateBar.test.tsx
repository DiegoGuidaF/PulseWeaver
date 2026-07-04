import { describe, expect, it, vi } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import { server } from "@/test/setup";
import { renderWithProviders, setupUser } from "@/test/utils";
import { policyAuditHandlers } from "@/test/mocks/handlers";
import { SimulateBar } from "./SimulateBar";
import { TEST_TIMEOUTS } from "@/test/constants";

// Pre-seed ip as a prop to avoid simulating parent rerender round-trips.
function renderBar(ip = "", onIpChange = vi.fn()) {
    return renderWithProviders(<SimulateBar ip={ip} onIpChange={onIpChange} />);
}

describe("SimulateBar", () => {
    it("Test button is disabled when both fields are empty", () => {
        renderBar();

        expect(screen.getByRole("button", { name: /test/i })).toBeDisabled();
    });

    it("Test button is disabled when IP is set but host is empty", () => {
        renderBar("192.168.1.10");

        // No host typed — canSubmit is false
        expect(screen.getByRole("button", { name: /test/i })).toBeDisabled();
    });

    it("Test button is enabled when both IP and host are filled", async () => {
        const user = setupUser();
        renderBar("192.168.1.10");

        await user.type(screen.getByRole("textbox", { name: /host \(fqdn\)/i }), "app.home.lan");

        expect(screen.getByRole("button", { name: /test/i })).not.toBeDisabled();
    });

    it("onIpChange is called when the IP field is edited", async () => {
        const user = setupUser();
        const onIpChange = vi.fn();
        renderBar("", onIpChange);

        await user.type(screen.getByRole("textbox", { name: /source ip/i }), "1");

        expect(onIpChange).toHaveBeenCalledWith("1");
    });

    it("shows Allowed alert after a successful simulation", async () => {
        const user = setupUser();
        server.use(
            policyAuditHandlers.simulate.allowed({
                ip: "192.168.1.10",
                host: "app.home.lan",
                allowed: true,
            }),
        );

        renderBar("192.168.1.10");
        await user.type(screen.getByRole("textbox", { name: /host \(fqdn\)/i }), "app.home.lan");
        await user.click(screen.getByRole("button", { name: /test/i }));

        await waitFor(
            () => expect(screen.getByText("Allowed")).toBeInTheDocument(),
            { timeout: TEST_TIMEOUTS.SHORT },
        );

        expect(screen.getByText(/192\.168\.1\.10.*app\.home\.lan/)).toBeInTheDocument();
    });

    it("shows Denied alert with ip_not_registered reason label", async () => {
        const user = setupUser();
        server.use(policyAuditHandlers.simulate.denied("ip_not_registered"));

        renderBar("1.2.3.4");
        await user.type(screen.getByRole("textbox", { name: /host \(fqdn\)/i }), "unknown.lan");
        await user.click(screen.getByRole("button", { name: /test/i }));

        await waitFor(
            () => expect(screen.getByText("Denied")).toBeInTheDocument(),
            { timeout: TEST_TIMEOUTS.SHORT },
        );

        expect(screen.getByText("IP not in any device or network policy")).toBeInTheDocument();
    });

    it("shows Denied alert with host_not_allowed reason label", async () => {
        const user = setupUser();
        server.use(policyAuditHandlers.simulate.denied("host_not_allowed"));

        renderBar("1.2.3.4");
        await user.type(screen.getByRole("textbox", { name: /host \(fqdn\)/i }), "blocked.lan");
        await user.click(screen.getByRole("button", { name: /test/i }));

        await waitFor(
            () => expect(screen.getByText("Denied")).toBeInTheDocument(),
            { timeout: TEST_TIMEOUTS.SHORT },
        );

        expect(screen.getByText("Host not allowed")).toBeInTheDocument();
    });

    it("shows a denied result when the response carries explicit nulls for all optional fields", async () => {
        const user = setupUser();
        // Real backend deny payload: match_source, network_policy_id and
        // network_policy_name are present as explicit JSON nulls, which the
        // SDK's Zod response validator must accept.
        server.use(
            policyAuditHandlers.simulate.denied("host_not_allowed", {
                ip: "62.151.105.183",
                host: "immich.wally.mywire.org",
                match_source: null,
                network_policy_id: null,
                network_policy_name: null,
            }),
        );

        renderBar("62.151.105.183");
        await user.type(
            screen.getByRole("textbox", { name: /host \(fqdn\)/i }),
            "immich.wally.mywire.org",
        );
        await user.click(screen.getByRole("button", { name: /test/i }));

        await waitFor(
            () => expect(screen.getByText("Denied")).toBeInTheDocument(),
            { timeout: TEST_TIMEOUTS.SHORT },
        );

        expect(screen.getByText("Host not allowed")).toBeInTheDocument();
    });

    it("shows an error alert when the simulation request fails", async () => {
        const user = setupUser();
        server.use(policyAuditHandlers.simulate.serverError());

        renderBar("192.168.1.10");
        await user.type(screen.getByRole("textbox", { name: /host \(fqdn\)/i }), "app.home.lan");
        await user.click(screen.getByRole("button", { name: /test/i }));

        await waitFor(
            () => expect(screen.getByText("Simulation failed")).toBeInTheDocument(),
            { timeout: TEST_TIMEOUTS.SHORT },
        );
    });

    it("result alert disappears after editing an input (dirty flag clears result)", async () => {
        const user = setupUser();
        server.use(policyAuditHandlers.simulate.allowed());

        renderBar("192.168.1.10");
        await user.type(screen.getByRole("textbox", { name: /host \(fqdn\)/i }), "app.home.lan");
        await user.click(screen.getByRole("button", { name: /test/i }));

        await waitFor(
            () => expect(screen.getByText("Allowed")).toBeInTheDocument(),
            { timeout: TEST_TIMEOUTS.SHORT },
        );

        // Edit the host field — dirty flag should hide the result
        await user.type(screen.getByRole("textbox", { name: /host \(fqdn\)/i }), "x");

        expect(screen.queryByText("Allowed")).not.toBeInTheDocument();
    });
});
