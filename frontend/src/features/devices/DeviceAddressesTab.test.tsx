import { beforeEach, describe, expect, it } from 'vitest';
import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { delay, http } from 'msw';
import { DeviceAddressesTab } from '@/features/devices/DeviceAddressesTab';
import { createMockAddress } from '@/test/mocks/data';
import { AddressEventSource } from '@/lib/api';
import { TEST_TIMEOUTS } from '@/test/constants';
import { addressHandlers, endpoints, responses } from '@/test/mocks/handlers';
import { server } from '@/test/setup';
import { renderWithProviders } from '@/test/utils';

const STALE_DATE = '2024-01-01T00:00:00Z'; // >7 days ago relative to test run date

function renderTab() {
    return renderWithProviders(<DeviceAddressesTab deviceId={1} />);
}

describe('DeviceAddressesTab', () => {
    beforeEach(() => {
        server.use(
            addressHandlers.list([
                createMockAddress({ ip: '10.0.0.5', is_enabled: true, source: AddressEventSource.HEARTBEAT }),
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

        expect(screen.queryByText('No active addresses.')).not.toBeInTheDocument();
        expect(screen.queryByText('10.0.0.5')).not.toBeInTheDocument();
    });

    it('shows empty state when no addresses', async () => {
        server.use(addressHandlers.list([]));

        renderTab();

        await waitFor(
            () => {
                expect(screen.getByText('No active addresses.')).toBeInTheDocument();
            },
            { timeout: TEST_TIMEOUTS.SHORT }
        );
        expect(screen.queryByRole('table')).not.toBeInTheDocument();
    });

    it('renders active address in table', async () => {
        renderTab();

        await waitFor(
            () => {
                expect(screen.getByText('10.0.0.5')).toBeInTheDocument();
            },
            { timeout: TEST_TIMEOUTS.SHORT }
        );
        expect(screen.getByText('live')).toBeInTheDocument();
        expect(screen.getByText(/heartbeat/)).toBeInTheDocument();
    });

    it('shows stale address in stale tab', async () => {
        const user = userEvent.setup();
        server.use(
            addressHandlers.list([
                createMockAddress({
                    id: 2,
                    ip: '10.0.0.99',
                    is_enabled: false,
                    updated_at: STALE_DATE,
                    created_at: STALE_DATE,
                    source: AddressEventSource.EXPIRY,
                }),
            ])
        );

        renderTab();

        await waitFor(
            () => {
                expect(screen.getByText(/Stale/)).toBeInTheDocument();
            },
            { timeout: TEST_TIMEOUTS.SHORT }
        );

        await user.click(screen.getByText(/Stale/));

        await waitFor(
            () => {
                expect(screen.getByText('10.0.0.99')).toBeInTheDocument();
            },
            { timeout: TEST_TIMEOUTS.SHORT }
        );
        expect(screen.getByRole('button', { name: 'Re-enable' })).toBeInTheDocument();
    });

    it('heartbeat registers IP and shows notification', async () => {
        const user = userEvent.setup();
        server.use(
            http.post(endpoints.deviceHeartbeat, async () => {
                await delay(50);
                return responses.ok(createMockAddress({ ip: '192.168.1.200', is_enabled: true, source: AddressEventSource.HEARTBEAT }));
            })
        );

        renderTab();

        await user.click(screen.getByRole('button', { name: /Register my IP/i }));

        await waitFor(
            () => {
                expect(screen.getByText('IP 192.168.1.200 registered')).toBeInTheDocument();
            },
            { timeout: TEST_TIMEOUTS.MEDIUM }
        );
    });

    it('can expand custom IP form and submit', async () => {
        const user = userEvent.setup();

        renderTab();

        await waitFor(
            () => {
                expect(screen.getByRole('button', { name: /Custom IP/i })).toBeInTheDocument();
            },
            { timeout: TEST_TIMEOUTS.SHORT }
        );

        await user.click(screen.getByRole('button', { name: /Custom IP/i }));

        const input = await screen.findByPlaceholderText('192.168.1.100 or 2001:db8::1');
        await user.type(input, '10.1.2.3');
        await user.click(screen.getByRole('button', { name: 'Add' }));

        await waitFor(
            () => {
                expect(screen.getByText('Address added')).toBeInTheDocument();
            },
            { timeout: TEST_TIMEOUTS.MEDIUM }
        );
    });

    it('add address error shows notification', async () => {
        const user = userEvent.setup();
        server.use(
            http.post(endpoints.deviceAddresses, () => responses.serverError())
        );

        renderTab();

        await waitFor(
            () => {
                expect(screen.getByRole('button', { name: /Custom IP/i })).toBeInTheDocument();
            },
            { timeout: TEST_TIMEOUTS.SHORT }
        );

        await user.click(screen.getByRole('button', { name: /Custom IP/i }));
        await user.type(screen.getByPlaceholderText('192.168.1.100 or 2001:db8::1'), '10.1.2.3');
        await user.click(screen.getByRole('button', { name: 'Add' }));

        await waitFor(
            () => {
                expect(screen.getByText('Error adding address')).toBeInTheDocument();
            },
            { timeout: TEST_TIMEOUTS.MEDIUM }
        );
    });
});
