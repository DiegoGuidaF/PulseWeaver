import { describe, expect, it } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import { delay, http } from "msw";
import { DeviceRulesTab } from "@/features/devices/DeviceRulesTab";
import { createMockDeviceAddressLeaseRule } from "@/test/mocks/data";
import { TEST_TIMEOUTS } from "@/test/constants";
import { endpoints, responses, ruleHandlers } from "@/test/mocks/handlers";
import { server } from "@/test/setup";
import { renderWithProviders, setupUser } from "@/test/utils";

function renderTab(liveAddressCount = 0) {
    return renderWithProviders(<DeviceRulesTab deviceId={1} liveAddressCount={liveAddressCount} />);
}

describe('DeviceRulesTab — Address lease rule', () => {
    it('shows loading skeleton', () => {
        server.use(
            http.get(endpoints.deviceAddressLeaseRule, async () => {
                await delay('infinite');
                return responses.ok(createMockDeviceAddressLeaseRule());
            })
        );

        renderTab();

        expect(screen.queryByRole('switch')).not.toBeInTheDocument();
    });

    it('shows disabled state with controls visible', async () => {
        server.use(ruleHandlers.addressLease.get.notFound());

        renderTab();

        await waitFor(
            () => {
                expect(screen.getAllByText('Disabled').length).toBeGreaterThanOrEqual(1);
            },
            { timeout: TEST_TIMEOUTS.SHORT }
        );
        // Controls are visible even when disabled (dimmed but interactive)
        expect(screen.getByRole('radio', { name: '1h' })).toBeInTheDocument();
        // No Save/Cancel shown — toggle is the enable action
        expect(screen.queryByRole('button', { name: 'Save' })).not.toBeInTheDocument();
    });

    it('shows enabled state with TTL selector', async () => {
        // defaultHandlers provides enabled=true, ttl_seconds=3600
        renderTab();

        await waitFor(
            () => {
                expect(screen.getAllByText('Enabled').length).toBeGreaterThanOrEqual(1);
            },
            { timeout: TEST_TIMEOUTS.SHORT }
        );
        expect(screen.getByRole('radio', { name: '1h' })).toBeChecked();
        // Save/Cancel not shown until user changes value
        expect(screen.queryByRole('button', { name: 'Save' })).not.toBeInTheDocument();
    });

    it('enables rule via toggle using the currently selected preset', async () => {
        const user = setupUser();
        server.use(ruleHandlers.addressLease.get.notFound());

        renderTab();

        await waitFor(
            () => {
                expect(screen.getByRole('switch', { name: /enable auto-expiry/i })).toBeInTheDocument();
            },
            { timeout: TEST_TIMEOUTS.SHORT }
        );

        // Change preset to 6h before enabling
        await user.click(screen.getByRole('radio', { name: '6h' }));
        await user.click(screen.getByRole('switch', { name: /enable auto-expiry/i }));

        await waitFor(
            () => {
                expect(screen.getByText('Address lease rule enabled')).toBeInTheDocument();
            },
            { timeout: TEST_TIMEOUTS.MEDIUM }
        );
    });

    it('shows Save/Cancel only after changing TTL preset when enabled', async () => {
        const user = setupUser();
        renderTab();

        await waitFor(
            () => {
                expect(screen.getByRole('radio', { name: '1h' })).toBeChecked();
            },
            { timeout: TEST_TIMEOUTS.SHORT }
        );
        expect(screen.queryByRole('button', { name: 'Save' })).not.toBeInTheDocument();

        await user.click(screen.getByRole('radio', { name: '6h' }));

        expect(screen.getByRole('button', { name: 'Save' })).toBeInTheDocument();
        expect(screen.getByRole('button', { name: 'Cancel' })).toBeInTheDocument();
    });

    it('saves changed TTL and shows toast', async () => {
        const user = setupUser();
        renderTab();

        await waitFor(
            () => {
                expect(screen.getByRole('radio', { name: '1h' })).toBeChecked();
            },
            { timeout: TEST_TIMEOUTS.SHORT }
        );
        await user.click(screen.getByRole('radio', { name: '6h' }));
        await user.click(screen.getByRole('button', { name: 'Save' }));

        await waitFor(
            () => {
                expect(screen.getByText('Address lease rule saved')).toBeInTheDocument();
            },
            { timeout: TEST_TIMEOUTS.MEDIUM }
        );
    });

    it('cancels TTL change and hides Save/Cancel', async () => {
        const user = setupUser();
        renderTab();

        await waitFor(
            () => {
                expect(screen.getByRole('radio', { name: '1h' })).toBeChecked();
            },
            { timeout: TEST_TIMEOUTS.SHORT }
        );
        await user.click(screen.getByRole('radio', { name: '6h' }));
        await user.click(screen.getByRole('button', { name: 'Cancel' }));

        expect(screen.queryByRole('button', { name: 'Cancel' })).not.toBeInTheDocument();
        expect(screen.queryByRole('button', { name: 'Save' })).not.toBeInTheDocument();
        expect(screen.getByRole('radio', { name: '1h' })).toBeChecked();
    });

    it('disables rule via toggle and shows toast', async () => {
        const user = setupUser();
        renderTab();

        await waitFor(
            () => {
                expect(screen.getByRole('switch', { name: /disable auto-expiry/i })).toBeInTheDocument();
            },
            { timeout: TEST_TIMEOUTS.MEDIUM }
        );
        await user.click(screen.getByRole('switch', { name: /disable auto-expiry/i }));

        await waitFor(
            () => {
                expect(screen.getByText('Address lease rule disabled')).toBeInTheDocument();
            },
            { timeout: TEST_TIMEOUTS.MEDIUM }
        );
    });

    it('shows error on fetch failure', async () => {
        server.use(
            http.get(endpoints.deviceAddressLeaseRule, () => responses.serverError())
        );

        renderTab();

        await waitFor(
            () => {
                expect(screen.getByText(/Error loading rule:/i)).toBeInTheDocument();
            },
            { timeout: TEST_TIMEOUTS.SHORT }
        );
    });
});

describe('DeviceRulesTab — Max active IPs rule', () => {
    it('shows disabled state with controls visible', async () => {
        server.use(ruleHandlers.maxActiveAddresses.get.notFound());
        renderTab();

        await waitFor(
            () => {
                expect(screen.getAllByText('Max active IPs').length).toBeGreaterThanOrEqual(1);
                expect(screen.getByRole('switch', { name: /enable max-ip rule/i })).toBeInTheDocument();
            },
            { timeout: TEST_TIMEOUTS.SHORT }
        );
        // Stepper controls always visible
        expect(screen.getByRole('button', { name: 'Decrease limit' })).toBeInTheDocument();
        expect(screen.getByRole('button', { name: 'Increase limit' })).toBeInTheDocument();
        // No Save/Cancel — toggle is the enable action
        expect(screen.queryByRole('button', { name: 'Save' })).not.toBeInTheDocument();
    });

    it('shows enabled state with limit controls', async () => {
        server.use(ruleHandlers.maxActiveAddresses.get.success({ max_addresses: 5 }));

        renderTab(2);

        await waitFor(
            () => {
                expect(screen.getByText('2/5')).toBeInTheDocument();
                expect(screen.getByRole('button', { name: 'Decrease limit' })).toBeInTheDocument();
            },
            { timeout: TEST_TIMEOUTS.SHORT }
        );
        // Save/Cancel not shown until user changes value
        expect(screen.queryByRole('button', { name: 'Save' })).not.toBeInTheDocument();
    });

    it('shows at-limit warning when live count meets limit', async () => {
        server.use(ruleHandlers.maxActiveAddresses.get.success({ max_addresses: 2 }));

        renderTab(2);

        await waitFor(
            () => {
                expect(screen.getByText('2/2')).toBeInTheDocument();
                expect(screen.getByText(/at limit/i)).toBeInTheDocument();
            },
            { timeout: TEST_TIMEOUTS.SHORT }
        );
    });

    it('shows eviction warning when limit is stepped below live count', async () => {
        const user = setupUser();
        server.use(ruleHandlers.maxActiveAddresses.get.success({ max_addresses: 5 }));

        renderTab(3); // 3 live addresses, limit currently 5

        await waitFor(
            () => {
                expect(screen.getByRole('button', { name: 'Decrease limit' })).toBeInTheDocument();
            },
            { timeout: TEST_TIMEOUTS.SHORT }
        );

        // Step limit down from 5 to 2 (below live count of 3)
        await user.click(screen.getByRole('button', { name: 'Decrease limit' }));
        await user.click(screen.getByRole('button', { name: 'Decrease limit' }));
        await user.click(screen.getByRole('button', { name: 'Decrease limit' }));

        expect(screen.getByText(/1 active address will be evicted/i)).toBeInTheDocument();
    });

    it('shows eviction warning in disabled state when limit would evict on enable', async () => {
        server.use(ruleHandlers.maxActiveAddresses.get.notFound());

        renderTab(3); // 3 live addresses, default limit 2

        await waitFor(
            () => {
                expect(screen.getByRole('button', { name: 'Decrease limit' })).toBeInTheDocument();
            },
            { timeout: TEST_TIMEOUTS.SHORT }
        );

        // Default is 2, live count is 3 → eviction warning immediately
        expect(screen.getByText(/1 active address will be evicted/i)).toBeInTheDocument();
    });

    it('enables rule via toggle using currently selected limit', async () => {
        const user = setupUser();
        server.use(ruleHandlers.maxActiveAddresses.get.notFound());

        renderTab();

        await waitFor(
            () => {
                expect(screen.getByRole('switch', { name: /enable max-ip rule/i })).toBeInTheDocument();
            },
            { timeout: TEST_TIMEOUTS.SHORT }
        );

        // Step up the limit before enabling
        await user.click(screen.getByRole('button', { name: 'Increase limit' }));
        await user.click(screen.getByRole('switch', { name: /enable max-ip rule/i }));

        await waitFor(
            () => {
                expect(screen.getByText('Max active IPs rule enabled')).toBeInTheDocument();
            },
            { timeout: TEST_TIMEOUTS.MEDIUM }
        );
    });

    it('shows Save/Cancel after stepping limit when enabled and saves', async () => {
        const user = setupUser();
        server.use(ruleHandlers.maxActiveAddresses.get.success({ max_addresses: 3 }));

        renderTab();

        await waitFor(
            () => {
                expect(screen.getByRole('button', { name: 'Increase limit' })).toBeInTheDocument();
            },
            { timeout: TEST_TIMEOUTS.SHORT }
        );
        expect(screen.queryByRole('button', { name: 'Save' })).not.toBeInTheDocument();

        await user.click(screen.getByRole('button', { name: 'Increase limit' }));
        await user.click(screen.getByRole('button', { name: 'Increase limit' }));

        expect(screen.getByRole('button', { name: 'Save' })).toBeInTheDocument();
        expect(screen.getByRole('button', { name: 'Cancel' })).toBeInTheDocument();

        await user.click(screen.getByRole('button', { name: 'Save' }));

        await waitFor(
            () => {
                expect(screen.getByText('Max active IPs rule saved')).toBeInTheDocument();
            },
            { timeout: TEST_TIMEOUTS.MEDIUM }
        );
    });

    it('cancels limit change and hides Save/Cancel', async () => {
        const user = setupUser();
        server.use(ruleHandlers.maxActiveAddresses.get.success());

        renderTab();

        await waitFor(
            () => {
                expect(screen.getByRole('button', { name: 'Increase limit' })).toBeInTheDocument();
            },
            { timeout: TEST_TIMEOUTS.SHORT }
        );
        await user.click(screen.getByRole('button', { name: 'Increase limit' }));
        await user.click(screen.getByRole('button', { name: 'Cancel' }));

        expect(screen.queryByRole('button', { name: 'Cancel' })).not.toBeInTheDocument();
        expect(screen.queryByRole('button', { name: 'Save' })).not.toBeInTheDocument();
    });

    it('disables rule via toggle and shows toast', async () => {
        const user = setupUser();
        server.use(ruleHandlers.maxActiveAddresses.get.success());

        renderTab();

        await waitFor(
            () => {
                expect(screen.getByRole('switch', { name: /disable max-ip rule/i })).toBeInTheDocument();
            },
            { timeout: TEST_TIMEOUTS.MEDIUM }
        );
        await user.click(screen.getByRole('switch', { name: /disable max-ip rule/i }));

        await waitFor(
            () => {
                expect(screen.getByText('Max active IPs rule disabled')).toBeInTheDocument();
            },
            { timeout: TEST_TIMEOUTS.MEDIUM }
        );
    });

    it('shows error on fetch failure', async () => {
        server.use(
            http.get(endpoints.maxActiveAddressesRule, () => responses.serverError())
        );

        renderTab();

        await waitFor(
            () => {
                expect(screen.getAllByText(/Error loading rule:/i).length).toBeGreaterThanOrEqual(1);
            },
            { timeout: TEST_TIMEOUTS.SHORT }
        );
    });
});
