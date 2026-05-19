import { beforeEach, describe, expect, it } from 'vitest';
import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { delay, http } from 'msw';
import { DeviceDetailPage } from '@/pages/devices/DeviceDetailPage';
import { createMockDevice } from '@/test/mocks/data';
import { TEST_TIMEOUTS } from '@/test/constants';
import { addressHandlers, deviceHandlers, endpoints, responses } from '@/test/mocks/handlers';
import { server } from '@/test/setup';
import { renderWithProviders } from '@/test/utils';

function renderPage(route = '/devices/1') {
    return renderWithProviders(<DeviceDetailPage />, {
        initialEntries: [route],
        path: '/devices/:deviceId',
    });
}

describe('DeviceDetailPage', () => {
    beforeEach(() => {
        server.use(
            deviceHandlers.getById({ name: 'My Router', api_key_prefix: 'rtr_' }),
            addressHandlers.list([]),
        );
    });

    it('redirects for non-numeric deviceId', async () => {
        renderPage('/devices/abc');

        await waitFor(
            () => {
                expect(screen.queryByText('My Router')).not.toBeInTheDocument();
            },
            { timeout: TEST_TIMEOUTS.SHORT }
        );
        expect(screen.queryByText(/API key prefix/i)).not.toBeInTheDocument();
        expect(screen.queryByRole('tab', { name: /addresses/i })).not.toBeInTheDocument();
        expect(screen.queryByRole('tab', { name: /settings/i })).not.toBeInTheDocument();
    });

    it('shows loading skeleton', () => {
        server.use(
            http.get(endpoints.deviceById, async () => {
                await delay('infinite');
                return responses.ok(createMockDevice({ name: 'My Router', api_key_prefix: 'rtr_' }));
            })
        );

        renderPage();

        expect(screen.queryByText('My Router')).not.toBeInTheDocument();
    });

    it('shows device header after load', async () => {
        renderPage();

        await waitFor(
            () => {
                expect(screen.getByRole('heading', { name: 'My Router' })).toBeInTheDocument();
            },
            { timeout: TEST_TIMEOUTS.SHORT }
        );
        expect(screen.getByRole('link', { name: /back to devices/i })).toBeInTheDocument();
    });

    it('shows error when device fetch fails', async () => {
        server.use(
            http.get(endpoints.deviceById, () => responses.serverError())
        );

        renderPage();

        await waitFor(
            () => {
                expect(screen.getByText(/Error loading device:/i)).toBeInTheDocument();
            },
            { timeout: TEST_TIMEOUTS.SHORT }
        );
    });

    it('switches to Rules tab', async () => {
        const user = userEvent.setup();

        renderPage();

        await waitFor(
            () => {
                expect(screen.getByRole('heading', { name: 'My Router' })).toBeInTheDocument();
            },
            { timeout: TEST_TIMEOUTS.SHORT }
        );
        await user.click(screen.getByRole('tab', { name: /rules/i }));

        await waitFor(
            () => {
                expect(screen.getByText('Auto-expiry rule')).toBeVisible();
            },
            { timeout: TEST_TIMEOUTS.SHORT }
        );
    });

    it('switches to Settings tab', async () => {
        const user = userEvent.setup();

        renderPage();

        await waitFor(
            () => {
                expect(screen.getByRole('heading', { name: 'My Router' })).toBeInTheDocument();
            },
            { timeout: TEST_TIMEOUTS.SHORT }
        );
        await user.click(screen.getByRole('tab', { name: /settings/i }));

        expect(screen.getByRole('button', { name: 'Regenerate API key' })).toBeVisible();
        expect(screen.queryByText('Register IP address')).not.toBeInTheDocument();
    });
});
