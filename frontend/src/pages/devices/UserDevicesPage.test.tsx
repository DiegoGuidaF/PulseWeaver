import { beforeEach, describe, expect, it } from 'vitest';
import { screen, waitFor } from '@testing-library/react';
import { delay, http } from 'msw';
import { UserDevicesPage } from '@/pages/devices/UserDevicesPage';
import {
    createMockFleetDevice,
    createMockOwnerFleetGroup,
} from '@/test/mocks/data';
import { RuleType } from '@/lib/api';
import { TEST_TIMEOUTS } from '@/test/constants';
import { endpoints, fleetHandlers, responses } from '@/test/mocks/handlers';
import { server } from '@/test/setup';
import { renderWithProviders, setupUser } from '@/test/utils';
import { ROUTES } from '@/lib/routes';

function renderPage(route = '/devices/owners/1?device=1') {
    return renderWithProviders(<UserDevicesPage />, {
        initialEntries: [route],
        path: ROUTES.userDevices,
    });
}

describe('UserDevicesPage', () => {
    beforeEach(() => {
        server.use(fleetHandlers.list());
    });

    it('redirects for non-numeric ownerId', async () => {
        renderPage('/devices/owners/abc');

        await waitFor(
            () => {
                expect(screen.queryByText('Test Device')).not.toBeInTheDocument();
            },
            { timeout: TEST_TIMEOUTS.SHORT }
        );
        expect(screen.queryByRole('tab', { name: /addresses/i })).not.toBeInTheDocument();
    });

    it('shows loading skeleton while fetching', () => {
        server.use(
            http.get(endpoints.deviceFleet, async () => {
                await delay('infinite');
                return responses.ok([]);
            })
        );

        renderPage();

        expect(screen.queryByText('Test Device')).not.toBeInTheDocument();
        expect(screen.queryByText('Test User')).not.toBeInTheDocument();
    });

    it('shows owner panel after load', async () => {
        renderPage();

        await waitFor(
            () => {
                expect(screen.getByText('Test User')).toBeInTheDocument();
            },
            { timeout: TEST_TIMEOUTS.SHORT }
        );
    });

    it('shows device name after load', async () => {
        renderPage();

        await waitFor(
            () => {
                expect(screen.getByRole('heading', { name: 'Test Device' })).toBeInTheDocument();
            },
            { timeout: TEST_TIMEOUTS.SHORT }
        );
    });

    describe('no-limits hint banner', () => {
        function renderWithRules(rules: { type: RuleType; enabled: boolean; limit?: number }[]) {
            server.use(
                fleetHandlers.list([
                    createMockOwnerFleetGroup({
                        devices: [createMockFleetDevice({ api_key_prefix: 'test_', rules })],
                    }),
                ]),
            );
            renderPage();
        }

        it('appears when the device has no configured rule', async () => {
            renderWithRules([]);

            await waitFor(
                () => expect(screen.getByText(/no address limits/i)).toBeInTheDocument(),
                { timeout: TEST_TIMEOUTS.SHORT }
            );
        });

        it('stays hidden when the device has an active limit rule', async () => {
            renderWithRules([{ type: RuleType.MAX_ACTIVE, enabled: true, limit: 3 }]);

            await waitFor(
                () => expect(screen.getByRole('heading', { name: 'Test Device' })).toBeInTheDocument(),
                { timeout: TEST_TIMEOUTS.SHORT }
            );
            expect(screen.queryByText(/no address limits/i)).not.toBeInTheDocument();
        });
    });

    // Zero groups means no such owner; one group with an empty devices list means a
    // real owner who has none yet. The two must not render the same thing.
    it('shows not-found in both panes when the owner resolves to no group', async () => {
        server.use(fleetHandlers.list([]));

        renderPage('/devices/owners/999?device=999');

        await waitFor(
            () => {
                expect(screen.getAllByText(/User not found/i)).toHaveLength(2);
            },
            { timeout: TEST_TIMEOUTS.SHORT }
        );
        expect(screen.queryByText(/No devices for/i)).not.toBeInTheDocument();
    });

    it('offers the create affordance for an owner with zero devices', async () => {
        server.use(fleetHandlers.list([createMockOwnerFleetGroup({ devices: [] })]));

        renderPage('/devices/owners/1');

        await waitFor(
            () => expect(screen.getByText(/No devices for/i)).toBeInTheDocument(),
            { timeout: TEST_TIMEOUTS.SHORT }
        );
        expect(screen.queryByText(/User not found/i)).not.toBeInTheDocument();
    });

    it('shows error alert when list fetch fails', async () => {
        server.use(
            http.get(endpoints.deviceFleet, () => responses.serverError())
        );

        renderPage();

        await waitFor(
            () => {
                expect(screen.getByText('Could not load devices')).toBeInTheDocument();
            },
            { timeout: TEST_TIMEOUTS.SHORT }
        );
    });

    it('auto-selects first device when no ?device= query param is present', async () => {
        renderPage('/devices/owners/1');

        await waitFor(
            () => {
                expect(screen.getByRole('heading', { name: 'Test Device' })).toBeInTheDocument();
            },
            { timeout: TEST_TIMEOUTS.SHORT }
        );
        expect(screen.getByRole('tab', { name: /addresses/i })).toBeInTheDocument();
    });

    it('switches to Settings tab and shows device profile card', async () => {
        const user = setupUser();

        renderPage();

        await waitFor(
            () => {
                expect(screen.getByRole('heading', { name: 'Test Device' })).toBeInTheDocument();
            },
            { timeout: TEST_TIMEOUTS.SHORT }
        );

        await user.click(screen.getByRole('tab', { name: /settings/i }));

        await waitFor(
            () => {
                expect(screen.getByRole('heading', { name: 'Profile' })).toBeInTheDocument();
            },
            { timeout: TEST_TIMEOUTS.SHORT }
        );
    });
});
