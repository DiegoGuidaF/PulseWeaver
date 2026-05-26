import { beforeEach, describe, expect, it } from 'vitest';
import { screen, waitFor } from '@testing-library/react';
import { DevicesPage } from '@/pages/devices/DevicesPage';
import { TEST_TIMEOUTS } from '@/test/constants';
import { deviceHandlers } from '@/test/mocks/handlers';
import { server } from '@/test/setup';
import { renderWithProviders } from '@/test/utils';
import { createMockDeviceOwnerGroup } from '@/test/mocks/data';

describe('DevicesPage', () => {
    beforeEach(() => {
        server.use(deviceHandlers.list([]));
    });

    it('renders the page heading', () => {
        renderWithProviders(<DevicesPage />);

        expect(screen.getByRole('heading', { name: 'Devices', level: 1 })).toBeInTheDocument();
    });

    it('shows empty state when no owner groups are returned', async () => {
        renderWithProviders(<DevicesPage />);

        await waitFor(
            () => {
                expect(screen.getByText('No devices found.')).toBeInTheDocument();
            },
            { timeout: TEST_TIMEOUTS.SHORT }
        );
    });

    it('renders owner name and device name from API response', async () => {
        server.use(
            deviceHandlers.list([
                createMockDeviceOwnerGroup({
                    owner: {
                        id: 1,
                        username: 'dguida',
                        display_name: 'Diego Guida',
                        role: 'user',
                        bypass_host_check: false,
                        host_groups: [],
                        device_count: 1,
                        live_address_count: 1,
                    },
                    devices: [
                        {
                            id: 10,
                            name: 'Diego Mac M1',
                            state: 'healthy',
                            live_address_count: 1,
                            rules: [],
                            created_at: '2024-01-01T00:00:00Z',
                        },
                    ],
                }),
            ])
        );

        renderWithProviders(<DevicesPage />);

        await waitFor(
            () => expect(screen.getByText('Diego Guida')).toBeInTheDocument(),
            { timeout: TEST_TIMEOUTS.SHORT }
        );

        expect(screen.getByText('Diego Mac M1')).toBeInTheDocument();
    });
});
