import { beforeEach, describe, expect, it, vi } from 'vitest';
import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Route, Routes } from 'react-router-dom';
import { ROUTES, buildRoute } from '@/lib/routes';
import { UserDetailPage } from '@/pages/access/users/UserDetailPage';
import { AuthProvider } from '@/features/auth/AuthContext';
import { TEST_TIMEOUTS } from '@/test/constants';
import { authHandlers, hostAccessHandlers } from '@/test/mocks/handlers';
import { server } from '@/test/setup';
import { renderWithProviders } from '@/test/utils';
import { createMockUserAccessDetail, createMockDeviceListItem } from '@/test/mocks/data';
import { UserRole } from '@/lib/api';

const { mockGuard } = vi.hoisted(() => ({ mockGuard: vi.fn() }));
vi.mock('@/hooks/useUnsavedChangesGuard', () => ({
    useUnsavedChangesGuard: (isDirty: boolean) => mockGuard(isDirty),
}));

function renderUserDetailPage(userId = 5) {
    return renderWithProviders(
        <AuthProvider><UserDetailPage /></AuthProvider>,
        { initialEntries: [`/access/users/${userId}`], path: '/access/users/:id' },
    );
}

function renderWithRoutes(userId = 5) {
    function TestApp() {
        return (
            <Routes>
                <Route path="/access/users/:id" element={<AuthProvider><UserDetailPage /></AuthProvider>} />
                <Route path={ROUTES.userDevices} element={<div data-testid="device-detail" />} />
            </Routes>
        );
    }
    return renderWithProviders(<TestApp />, { initialEntries: [`/access/users/${userId}`] });
}

describe('UserDetailPage', () => {
    beforeEach(() => mockGuard.mockClear());

    describe('devices tab', () => {
        beforeEach(() => {
            server.use(
                authHandlers.me.success({ id: 99, username: 'admin', role: UserRole.ADMIN }),
                hostAccessHandlers.userHostDetails.success(
                    createMockUserAccessDetail({
                        id: 5,
                        username: 'charlie',
                        display_name: 'Charlie',
                        role: UserRole.USER,
                        devices: [
                            createMockDeviceListItem({ id: 10, name: 'Charlie Laptop' }),
                        ],
                    }),
                ),
            );
        });

        it('clicking a device row navigates to the device detail page', async () => {
            const user = userEvent.setup();

            renderWithRoutes();

            await waitFor(
                () => expect(screen.getByText('Charlie Laptop')).toBeInTheDocument(),
                { timeout: TEST_TIMEOUTS.SHORT },
            );

            await user.click(screen.getByText('Charlie Laptop'));

            await waitFor(
                () => expect(screen.getByTestId('device-detail')).toBeInTheDocument(),
                { timeout: TEST_TIMEOUTS.SHORT },
            );
        });

        it('"All devices →" link points to the owner device workspace', async () => {
            const user = userEvent.setup();
            renderUserDetailPage(5);

            // Wait for data, then activate the Devices tab so the panel leaves aria-hidden
            await waitFor(
                () => expect(screen.getByText('Charlie Laptop')).toBeInTheDocument(),
                { timeout: TEST_TIMEOUTS.SHORT },
            );
            await user.click(screen.getByRole('tab', { name: /devices/i }));

            const allDevicesLink = screen.getByRole('link', { name: /all devices/i });
            expect(allDevicesLink).toHaveAttribute('href', buildRoute.userDevices(5));
        });
    });
});
