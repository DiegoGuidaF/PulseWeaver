import { beforeEach, describe, expect, it } from 'vitest';
import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { delay, http } from 'msw';
import { UserDeviceWorkspacePage } from '@/pages/devices/UserDeviceWorkspacePage';
import { createMockDeviceOwnerGroup } from '@/test/mocks/data';
import { TEST_TIMEOUTS } from '@/test/constants';
import { deviceHandlers, endpoints, responses } from '@/test/mocks/handlers';
import { server } from '@/test/setup';
import { renderWithProviders } from '@/test/utils';

function renderPage(route = '/user-devices/1?device=1') {
    return renderWithProviders(<UserDeviceWorkspacePage />, {
        initialEntries: [route],
        path: '/user-devices/:userId',
    });
}

describe('UserDeviceWorkspacePage', () => {
    beforeEach(() => {
        server.use(
            deviceHandlers.list([
                createMockDeviceOwnerGroup(),
            ])
        );
    });

    it('redirects for non-numeric userId', async () => {
        renderPage('/user-devices/abc');

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
            http.get(endpoints.devices, async () => {
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

    it('shows not-found message when owner is not in list', async () => {
        server.use(deviceHandlers.list([]));

        renderPage('/user-devices/999?device=999');

        await waitFor(
            () => {
                expect(screen.getByText(/User not found/i)).toBeInTheDocument();
            },
            { timeout: TEST_TIMEOUTS.SHORT }
        );
    });

    it('shows error alert when list fetch fails', async () => {
        server.use(
            http.get(endpoints.devices, () => responses.serverError())
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
        renderPage('/user-devices/1');

        await waitFor(
            () => {
                expect(screen.getByRole('heading', { name: 'Test Device' })).toBeInTheDocument();
            },
            { timeout: TEST_TIMEOUTS.SHORT }
        );
        expect(screen.getByRole('tab', { name: /addresses/i })).toBeInTheDocument();
    });

    it('switches to Settings tab and shows device profile card', async () => {
        const user = userEvent.setup();

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
                expect(screen.getByRole('heading', { name: 'Device profile' })).toBeInTheDocument();
            },
            { timeout: TEST_TIMEOUTS.SHORT }
        );
    });
});
