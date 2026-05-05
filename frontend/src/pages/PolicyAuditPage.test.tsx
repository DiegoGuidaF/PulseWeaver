import { describe, expect, it } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { server } from "@/test/setup";
import { renderWithProviders } from "@/test/utils";
import { policyAuditHandlers, responses, endpoints } from "@/test/mocks/handlers";
import { createMockPolicyUserEntry, createMockPolicyUserIp } from "@/test/mocks/data";
import { PolicyAuditPage } from "./PolicyAuditPage";
import { http } from "msw";
import { TEST_TIMEOUTS } from "@/test/constants";

describe("PolicyAuditPage", () => {
    it("renders stats header with cache metadata", async () => {
        server.use(
            policyAuditHandlers.policyMap.success({
                total_ip_count: 7,
                total_device_count: 4,
                total_host_count: 12,
                shared_ip_count: 2,
            }),
        );

        renderWithProviders(<PolicyAuditPage />);

        await waitFor(
            () => expect(screen.getByText("7")).toBeInTheDocument(),
            { timeout: TEST_TIMEOUTS.SHORT },
        );

        expect(screen.getByText("4 devices")).toBeInTheDocument();
        expect(screen.getByText("12")).toBeInTheDocument();
        expect(screen.getByText("2")).toBeInTheDocument();
        // relativeTime is time-sensitive — match loosely
        expect(screen.getByText(/\d+[smh] ago/)).toBeInTheDocument();
        expect(screen.getByText("42ms")).toBeInTheDocument();
    });

    it("shows error alert when API returns 500", async () => {
        server.use(policyAuditHandlers.policyMap.serverError());

        renderWithProviders(<PolicyAuditPage />);

        await waitFor(
            () =>
                expect(
                    screen.getByText("Failed to load policy cache"),
                ).toBeInTheDocument(),
            { timeout: TEST_TIMEOUTS.SHORT },
        );
    });

    it("shows error alert when API returns 403", async () => {
        server.use(
            http.get(endpoints.policyMap, () => responses.forbidden()),
        );

        renderWithProviders(<PolicyAuditPage />);

        await waitFor(
            () =>
                expect(
                    screen.getByText("Failed to load policy cache"),
                ).toBeInTheDocument(),
            { timeout: TEST_TIMEOUTS.SHORT },
        );
    });

    it("renders user rows from the policy map", async () => {
        renderWithProviders(<PolicyAuditPage />);

        // default mock has alice / bob / carol
        await waitFor(
            () => expect(screen.getByText("alice")).toBeInTheDocument(),
            { timeout: TEST_TIMEOUTS.SHORT },
        );

        expect(screen.getByText("bob")).toBeInTheDocument();
        expect(screen.getByText("carol")).toBeInTheDocument();
    });

    it("clicking a user row opens the drawer", async () => {
        const user = userEvent.setup();
        renderWithProviders(<PolicyAuditPage />);

        await waitFor(
            () => expect(screen.getByText("alice")).toBeInTheDocument(),
            { timeout: TEST_TIMEOUTS.SHORT },
        );

        await user.click(screen.getByText("alice"));

        await waitFor(
            () =>
                expect(
                    screen.getByText("User · Policy"),
                ).toBeInTheDocument(),
            { timeout: TEST_TIMEOUTS.SHORT },
        );
    });

    it("clicking an IP badge populates the Source IP field in SimulateBar", async () => {
        const user = userEvent.setup();
        server.use(
            policyAuditHandlers.policyMap.success({
                users: [
                    createMockPolicyUserEntry({
                        user_id: 1,
                        user_name: "alice",
                        ips: [createMockPolicyUserIp({ ip: "10.0.1.99" })],
                    }),
                ],
            }),
        );

        renderWithProviders(<PolicyAuditPage />);

        await waitFor(
            () => expect(screen.getByText(/10\.0\.1\.99/)).toBeInTheDocument(),
            { timeout: TEST_TIMEOUTS.SHORT },
        );

        await user.click(screen.getByText(/10\.0\.1\.99/));

        const ipInput = screen.getByRole("textbox", { name: /source ip/i });
        expect(ipInput).toHaveValue("10.0.1.99");
    });
});
