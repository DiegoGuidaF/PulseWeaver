import { beforeEach, describe, expect, it } from 'vitest';
import { screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { delay, http } from 'msw';
import { DeviceAddressesTab } from '@/features/devices/DeviceAddressesTab';
import { createMockAddress } from '@/test/mocks/data';
import { TEST_TIMEOUTS } from '@/test/constants';
import { addressHandlers, endpoints, responses } from '@/test/mocks/handlers';
import { server } from '@/test/setup';
import { renderWithProviders } from '@/test/utils';

function renderTab() {
    return renderWithProviders(<DeviceAddressesTab deviceId={1} />);
}

describe('DeviceAddressesTab', () => {
    beforeEach(() => {
        // Override default address list with specific IP used by most tests
        server.use(
            addressHandlers.list([
                createMockAddress({ ip: '10.0.0.5', is_enabled: true }),
            ])
        );
    });

    it('shows loading skeleton while fetching', () => {
        server.use(
            http.get(endpoints.deviceAddresses, async () => {
                await delay('infinite');
                return responses.ok([]);
            })
        );

        renderTab();

        expect(screen.queryByText('No addresses assigned yet.')).not.toBeInTheDocument();
        expect(screen.queryByText('10.0.0.5')).not.toBeInTheDocument();
    });

    it('shows empty state', async () => {
        server.use(addressHandlers.list([]));

        renderTab();

        await waitFor(
            () => {
                expect(screen.getByText('No addresses assigned yet.')).toBeInTheDocument();
            },
            { timeout: TEST_TIMEOUTS.SHORT }
        );
        expect(screen.queryByRole('table')).not.toBeInTheDocument();
    });

    it('renders active address with Disable button', async () => {
        renderTab();

        await waitFor(
            () => {
                expect(screen.getByText('10.0.0.5')).toBeInTheDocument();
            },
            { timeout: TEST_TIMEOUTS.SHORT }
        );
        expect(screen.getByText('Active')).toBeInTheDocument();
        expect(screen.getByRole('button', { name: 'Disable' })).toBeInTheDocument();
        expect(screen.queryByRole('button', { name: 'Enable' })).not.toBeInTheDocument();
    });

    it('renders inactive address with Enable button', async () => {
        server.use(
            addressHandlers.list([
                createMockAddress({ ip: '10.0.0.5', is_enabled: false }),
            ])
        );

        renderTab();

        await waitFor(
            () => {
                expect(screen.getByText('Inactive')).toBeInTheDocument();
            },
            { timeout: TEST_TIMEOUTS.SHORT }
        );
        expect(screen.getByRole('button', { name: 'Enable' })).toBeInTheDocument();
        expect(screen.queryByRole('button', { name: 'Disable' })).not.toBeInTheDocument();
    });

    it('opens disable confirmation dialog', async () => {
        const user = userEvent.setup();
        renderTab();

        await waitFor(
            () => {
                expect(screen.getByText('10.0.0.5')).toBeInTheDocument();
            },
            { timeout: TEST_TIMEOUTS.SHORT }
        );
        await user.click(screen.getByRole('button', { name: 'Disable' }));

        const dialog = screen.getByRole('dialog', { name: /disable address/i });
        expect(dialog).toBeInTheDocument();
        expect(within(dialog).getByText('10.0.0.5')).toBeInTheDocument();
    });

    it('confirms disable and shows toast', async () => {
        const user = userEvent.setup();
        // addressHandlers.disable.success() is in defaultHandlers

        renderTab();

        await waitFor(
            () => {
                expect(screen.getByText('10.0.0.5')).toBeInTheDocument();
            },
            { timeout: TEST_TIMEOUTS.SHORT }
        );
        await user.click(screen.getByRole('button', { name: 'Disable' }));
        const dialog = screen.getByRole('dialog', { name: /disable address/i });
        await user.click(within(dialog).getByRole('button', { name: 'Disable' }));

        await waitFor(
            () => {
                expect(screen.getByText('Address disabled')).toBeInTheDocument();
            },
            { timeout: TEST_TIMEOUTS.MEDIUM }
        );
    });

    it('heartbeat registers IP and shows result', async () => {
        const user = userEvent.setup();
        server.use(
            http.post(endpoints.deviceHeartbeat, async () => {
                await delay(100);
                return responses.ok(createMockAddress({ ip: '192.168.1.100', is_enabled: true }));
            })
        );

        renderTab();

        await user.click(screen.getByRole('button', { name: /register my ip/i }));
        expect(screen.getByRole('button', { name: /registering/i })).toBeDisabled();

        await waitFor(
            () => {
                expect(screen.getByText('Your current IP:')).toBeInTheDocument();
                expect(screen.getByText('192.168.1.100')).toBeInTheDocument();
                expect(screen.getByText('IP 192.168.1.100 registered')).toBeInTheDocument();
            },
            { timeout: TEST_TIMEOUTS.MEDIUM }
        );
    });

    it('heartbeat error shows toast', async () => {
        const user = userEvent.setup();
        server.use(
            http.post(endpoints.deviceHeartbeat, () => responses.serverError())
        );

        renderTab();

        await user.click(screen.getByRole('button', { name: /register my ip/i }));

        await waitFor(
            () => {
                expect(screen.getByText('Heartbeat failed')).toBeInTheDocument();
            },
            { timeout: TEST_TIMEOUTS.MEDIUM }
        );
    });

    it('adds an address and resets form', async () => {
        const user = userEvent.setup();
        server.use(addressHandlers.list([]));
        // addressHandlers.create.success() is in defaultHandlers

        renderTab();

        const input = screen.getByRole('textbox', { name: /ip address/i });
        await user.type(input, '10.0.1.1');
        await user.click(screen.getByRole('button', { name: /add ip/i }));

        await waitFor(
            () => {
                expect(screen.getByText('Address added')).toBeInTheDocument();
            },
            { timeout: TEST_TIMEOUTS.MEDIUM }
        );
        expect(input).toHaveValue('');
    });

    it('add address error shows toast', async () => {
        const user = userEvent.setup();
        server.use(
            addressHandlers.list([]),
            http.post(endpoints.deviceAddresses, () => responses.serverError())
        );

        renderTab();

        await user.type(screen.getByRole('textbox', { name: /ip address/i }), '10.0.1.1');
        await user.click(screen.getByRole('button', { name: /add ip/i }));

        await waitFor(
            () => {
                expect(screen.getByText('Error adding address')).toBeInTheDocument();
            },
            { timeout: TEST_TIMEOUTS.MEDIUM }
        );
    });
});
