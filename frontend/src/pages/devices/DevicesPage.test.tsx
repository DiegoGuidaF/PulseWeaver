import { beforeEach, describe, expect, it } from 'vitest';
import { screen, waitFor } from '@testing-library/react';
import { DevicesPage } from '@/pages/devices/DevicesPage';
import { TEST_TIMEOUTS } from '@/test/constants';
import { fleetHandlers } from '@/test/mocks/handlers';
import { server } from '@/test/setup';
import { renderWithProviders } from '@/test/utils';
import {
    createMockFleetDevice,
    createMockOwnerFleetGroup,
    createMockOwnerSummary,
} from '@/test/mocks/data';
import { DevicePairingStatus, RuleType } from '@/lib/api';
import dayjs from 'dayjs';

describe('DevicesPage', () => {
    beforeEach(() => {
        server.use(fleetHandlers.list([]));
    });

    it('renders the page heading', () => {
        renderWithProviders(<DevicesPage />);

        expect(screen.getByRole('heading', { name: 'Devices', level: 1 })).toBeInTheDocument();
    });

    it('shows empty state when no owners are returned', async () => {
        renderWithProviders(<DevicesPage />);

        await waitFor(
            () => {
                expect(screen.getByText('No devices yet')).toBeInTheDocument();
            },
            { timeout: TEST_TIMEOUTS.SHORT }
        );
    });

    it('renders owner name and aggregates from API response', async () => {
        server.use(
            fleetHandlers.list([
                createMockOwnerFleetGroup({
                    owner: createMockOwnerSummary({
                        id: 1,
                        display_name: 'Diego Guida',
                        host_groups: [],
                        device_count: 1,
                        live_address_count: 1,
                    }),
                }),
            ])
        );

        renderWithProviders(<DevicesPage />);

        await waitFor(
            () => expect(screen.getByText('Diego Guida')).toBeInTheDocument(),
            { timeout: TEST_TIMEOUTS.SHORT }
        );

        expect(screen.getByText(/1 device/)).toBeInTheDocument();
        expect(screen.getByText(/1 IPs live/)).toBeInTheDocument();
    });

    it('renders each owner device row with its rule chip', async () => {
        server.use(
            fleetHandlers.list([
                createMockOwnerFleetGroup({
                    owner: createMockOwnerSummary({ id: 1, display_name: 'Diego Guida' }),
                    devices: [
                        createMockFleetDevice({
                            id: 7,
                            name: 'Work Laptop',
                            live_address_count: 1,
                            rules: [{ type: RuleType.MAX_ACTIVE, enabled: true, limit: 3 }],
                        }),
                    ],
                }),
            ])
        );

        renderWithProviders(<DevicesPage />);

        expect(
            await screen.findByText('Work Laptop', {}, { timeout: TEST_TIMEOUTS.SHORT })
        ).toBeInTheDocument();

        await waitFor(
            () => expect(screen.getByLabelText(/Max active IPs · 1 of 3/)).toBeInTheDocument(),
            { timeout: TEST_TIMEOUTS.SHORT }
        );
        expect(screen.getByText('1/3')).toBeInTheDocument();
    });

    it('renders the pairing chip instead of rule chips while a pairing is outstanding', async () => {
        server.use(
            fleetHandlers.list([
                createMockOwnerFleetGroup({
                    owner: createMockOwnerSummary({ id: 1, display_name: 'Diego Guida' }),
                    devices: [
                        createMockFleetDevice({
                            id: 7,
                            name: 'Work Laptop',
                            pairing: {
                                status: DevicePairingStatus.PENDING,
                                expires_at: dayjs().add(45, 'minute').toISOString(),
                            },
                        }),
                    ],
                }),
            ])
        );

        renderWithProviders(<DevicesPage />);

        await waitFor(
            () => expect(screen.getByLabelText(/Pairing pending/)).toBeInTheDocument(),
            { timeout: TEST_TIMEOUTS.SHORT }
        );
    });
});
