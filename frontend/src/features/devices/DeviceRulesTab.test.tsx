import { describe, expect, it } from 'vitest';
import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { delay, http } from 'msw';
import { DeviceRulesTab } from '@/features/devices/DeviceRulesTab';
import { createMockDeviceAddressLeaseRule } from '@/test/mocks/data';
import { TEST_TIMEOUTS } from '@/test/constants';
import { endpoints, responses, ruleHandlers } from '@/test/mocks/handlers';
import { server } from '@/test/setup';
import { renderWithProviders } from '@/test/utils';

function renderTab() {
    return renderWithProviders(<DeviceRulesTab deviceId={1} />);
}

describe('DeviceRulesTab', () => {
    it('shows loading skeleton', () => {
        server.use(
            http.get(endpoints.deviceAddressLeaseRule, async () => {
                await delay('infinite');
                return responses.ok(createMockDeviceAddressLeaseRule());
            })
        );

        renderTab();

        expect(screen.queryByText('Enabled')).not.toBeInTheDocument();
        expect(screen.queryByText(/disabled/i)).not.toBeInTheDocument();
    });

    it('shows disabled state when no rule (404)', async () => {
        server.use(ruleHandlers.addressLease.get.notFound());

        renderTab();

        await waitFor(
            () => {
                expect(screen.getAllByText('Status:').length).toBeGreaterThanOrEqual(1);
                expect(screen.getAllByText('Disabled').length).toBeGreaterThanOrEqual(1);
            },
            { timeout: TEST_TIMEOUTS.SHORT }
        );
        expect(screen.getByRole('button', { name: 'Enable auto-expiry' })).toBeInTheDocument();
        expect(screen.queryByRole('button', { name: 'Turn off auto-expiry' })).not.toBeInTheDocument();
        expect(screen.getByRole('spinbutton', { name: /expires after/i })).toHaveValue(5);
        expect(screen.getByRole('combobox', { name: /unit/i })).toHaveValue('minutes');
    });

    it('shows enabled state with TTL', async () => {
        // defaultHandlers provides enabled=true, ttl_seconds=3600 for lease; max_addresses=notFound
        renderTab();

        await waitFor(
            () => {
                expect(screen.getAllByText('Status:').length).toBeGreaterThanOrEqual(1);
                expect(screen.getByText('Enabled')).toBeInTheDocument();
            },
            { timeout: TEST_TIMEOUTS.SHORT }
        );
        expect(screen.getByText('1 hour')).toBeInTheDocument();
        expect(screen.getByRole('button', { name: 'Change TTL' })).toBeInTheDocument();
        expect(screen.getByRole('button', { name: 'Turn off auto-expiry' })).toBeInTheDocument();
        expect(screen.queryByRole('button', { name: 'Save' })).not.toBeInTheDocument();
    });

    it('enables auto-expiry and shows toast', async () => {
        const user = userEvent.setup();
        server.use(ruleHandlers.addressLease.get.notFound());
        // ruleHandlers.addressLease.put.success() is in defaultHandlers

        renderTab();

        await waitFor(
            () => {
                expect(screen.getByRole('button', { name: 'Enable auto-expiry' })).toBeInTheDocument();
            },
            { timeout: TEST_TIMEOUTS.SHORT }
        );
        await user.click(screen.getByRole('button', { name: 'Enable auto-expiry' }));

        await waitFor(
            () => {
                expect(screen.getByText('Address lease rule saved')).toBeInTheDocument();
            },
            { timeout: TEST_TIMEOUTS.MEDIUM }
        );
    });

    it('edits TTL and shows toast', async () => {
        const user = userEvent.setup();
        // ruleHandlers.addressLease.put.success() is in defaultHandlers

        renderTab();

        await waitFor(
            () => {
                expect(screen.getByRole('button', { name: 'Change TTL' })).toBeInTheDocument();
            },
            { timeout: TEST_TIMEOUTS.SHORT }
        );
        await user.click(screen.getByRole('button', { name: 'Change TTL' }));

        const valueInput = screen.getByRole('spinbutton', { name: /expires after/i });
        await user.clear(valueInput);
        await user.type(valueInput, '2');
        await user.selectOptions(screen.getByRole('combobox', { name: /unit/i }), 'days');
        await user.click(screen.getByRole('button', { name: 'Save' }));

        await waitFor(
            () => {
                expect(screen.getByText('Address lease rule saved')).toBeInTheDocument();
            },
            { timeout: TEST_TIMEOUTS.MEDIUM }
        );
        expect(screen.queryByRole('button', { name: 'Cancel' })).not.toBeInTheDocument();
    });

    it('cancels TTL edit', async () => {
        const user = userEvent.setup();
        renderTab();

        await waitFor(
            () => {
                expect(screen.getByRole('button', { name: 'Change TTL' })).toBeInTheDocument();
            },
            { timeout: TEST_TIMEOUTS.SHORT }
        );
        await user.click(screen.getByRole('button', { name: 'Change TTL' }));
        await user.click(screen.getByRole('button', { name: 'Cancel' }));

        expect(screen.queryByRole('button', { name: 'Cancel' })).not.toBeInTheDocument();
        expect(screen.getByRole('button', { name: 'Change TTL' })).toBeInTheDocument();
        expect(screen.getByRole('button', { name: 'Turn off auto-expiry' })).toBeInTheDocument();
        expect(screen.queryByText('Address lease rule saved')).not.toBeInTheDocument();
    });

    it('turns off auto-expiry', async () => {
        const user = userEvent.setup();
        // ruleHandlers.addressLease.delete.success() is in defaultHandlers

        renderTab();

        await waitFor(
            () => {
                expect(screen.getByRole('button', { name: 'Turn off auto-expiry' })).toBeInTheDocument();
            },
            { timeout: TEST_TIMEOUTS.MEDIUM }
        );
        await user.click(screen.getByRole('button', { name: 'Turn off auto-expiry' }));

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
    it('shows disabled state when no rule (404)', async () => {
        // defaultHandlers already returns notFound for max_active_addresses
        renderTab();

        await waitFor(
            () => {
                expect(screen.getByText('Max active IPs rule')).toBeInTheDocument();
                expect(screen.getByRole('button', { name: 'Enable max-IP rule' })).toBeInTheDocument();
            },
            { timeout: TEST_TIMEOUTS.SHORT }
        );
        expect(screen.queryByRole('button', { name: 'Turn off max-IP rule' })).not.toBeInTheDocument();
        expect(screen.getByRole('spinbutton', { name: /max active ips/i })).toHaveValue(2);
    });

    it('shows enabled state with max_addresses value', async () => {
        server.use(ruleHandlers.maxActiveAddresses.get.success({ max_addresses: 5 }));

        renderTab();

        await waitFor(
            () => {
                expect(screen.getByText('5')).toBeInTheDocument();
                expect(screen.getByRole('button', { name: 'Change limit' })).toBeInTheDocument();
            },
            { timeout: TEST_TIMEOUTS.SHORT }
        );
        expect(screen.getByRole('button', { name: 'Turn off max-IP rule' })).toBeInTheDocument();
        expect(screen.queryByRole('button', { name: 'Enable max-IP rule' })).not.toBeInTheDocument();
    });

    it('enables rule and shows toast', async () => {
        const user = userEvent.setup();
        // defaultHandlers: notFound for get, success for put

        renderTab();

        await waitFor(
            () => {
                expect(screen.getByRole('button', { name: 'Enable max-IP rule' })).toBeInTheDocument();
            },
            { timeout: TEST_TIMEOUTS.SHORT }
        );
        await user.click(screen.getByRole('button', { name: 'Enable max-IP rule' }));

        await waitFor(
            () => {
                expect(screen.getByText('Max active IPs rule saved')).toBeInTheDocument();
            },
            { timeout: TEST_TIMEOUTS.MEDIUM }
        );
    });

    it('edits limit and shows toast', async () => {
        const user = userEvent.setup();
        server.use(ruleHandlers.maxActiveAddresses.get.success({ max_addresses: 3 }));

        renderTab();

        await waitFor(
            () => {
                expect(screen.getByRole('button', { name: 'Change limit' })).toBeInTheDocument();
            },
            { timeout: TEST_TIMEOUTS.SHORT }
        );
        await user.click(screen.getByRole('button', { name: 'Change limit' }));

        const input = screen.getByRole('spinbutton', { name: /max active ips/i });
        await user.clear(input);
        await user.type(input, '5');
        await user.click(screen.getByRole('button', { name: 'Save' }));

        await waitFor(
            () => {
                expect(screen.getByText('Max active IPs rule saved')).toBeInTheDocument();
            },
            { timeout: TEST_TIMEOUTS.MEDIUM }
        );
        expect(screen.queryByRole('button', { name: 'Cancel' })).not.toBeInTheDocument();
    });

    it('cancels limit edit', async () => {
        const user = userEvent.setup();
        server.use(ruleHandlers.maxActiveAddresses.get.success());

        renderTab();

        await waitFor(
            () => {
                expect(screen.getByRole('button', { name: 'Change limit' })).toBeInTheDocument();
            },
            { timeout: TEST_TIMEOUTS.SHORT }
        );
        await user.click(screen.getByRole('button', { name: 'Change limit' }));
        await user.click(screen.getByRole('button', { name: 'Cancel' }));

        expect(screen.queryByRole('button', { name: 'Cancel' })).not.toBeInTheDocument();
        expect(screen.getByRole('button', { name: 'Change limit' })).toBeInTheDocument();
        expect(screen.getByRole('button', { name: 'Turn off max-IP rule' })).toBeInTheDocument();
    });

    it('turns off rule and shows toast', async () => {
        const user = userEvent.setup();
        server.use(ruleHandlers.maxActiveAddresses.get.success());

        renderTab();

        await waitFor(
            () => {
                expect(screen.getByRole('button', { name: 'Turn off max-IP rule' })).toBeInTheDocument();
            },
            { timeout: TEST_TIMEOUTS.MEDIUM }
        );
        await user.click(screen.getByRole('button', { name: 'Turn off max-IP rule' }));

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
