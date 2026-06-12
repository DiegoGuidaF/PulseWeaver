import { beforeEach, describe, expect, it, vi } from 'vitest';
import { screen, waitFor } from '@testing-library/react';
import { Route, Routes } from 'react-router-dom';
import { ROUTES, buildRoute } from '@/lib/routes';
import { UserDetailPage } from '@/pages/access/users/UserDetailPage';
import { AuthProvider } from '@/features/auth/AuthContext';
import { TEST_TIMEOUTS } from '@/test/constants';
import { authHandlers, hostAccessHandlers } from '@/test/mocks/handlers';
import { server } from '@/test/setup';
import { renderWithProviders, setupUser } from '@/test/utils';
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
            const user = setupUser();

            renderWithRoutes();

            await waitFor(
                () => expect(screen.getByText('Charlie Laptop')).toBeInTheDocument(),
                { timeout: TEST_TIMEOUTS.SHORT },
            );

            await user.click(screen.getByRole('tab', { name: /devices/i }));
            await user.click(screen.getByText('Charlie Laptop'));

            await waitFor(
                () => expect(screen.getByTestId('device-detail')).toBeInTheDocument(),
                { timeout: TEST_TIMEOUTS.SHORT },
            );
        });

        it('"Manage all devices →" link points to the owner device workspace', async () => {
            const user = setupUser();
            renderUserDetailPage(5);

            // Wait for data, then activate the Devices tab so the panel leaves aria-hidden
            await waitFor(
                () => expect(screen.getByText('Charlie Laptop')).toBeInTheDocument(),
                { timeout: TEST_TIMEOUTS.SHORT },
            );
            await user.click(screen.getByRole('tab', { name: /devices/i }));

            const allDevicesLink = screen.getByRole('link', { name: /manage all devices/i });
            expect(allDevicesLink).toHaveAttribute('href', buildRoute.userDevices(5));
        });

        it('opens the Devices tab when ?tab=devices is in the URL', async () => {
            renderWithProviders(
                <AuthProvider><UserDetailPage /></AuthProvider>,
                { initialEntries: ['/access/users/5?tab=devices'], path: '/access/users/:id' },
            );

            await waitFor(
                () => expect(screen.getByText('Charlie Laptop')).toBeInTheDocument(),
                { timeout: TEST_TIMEOUTS.SHORT },
            );

            expect(screen.getByRole('tab', { name: /devices/i })).toHaveAttribute(
                'aria-selected',
                'true',
            );
        });
    });

    describe('bypass acknowledge gate', () => {
        beforeEach(() => {
            server.use(authHandlers.me.success({ id: 99, username: 'admin', role: UserRole.ADMIN }));
        });

        it('switches the save bar to a warning and disables Save until acknowledged on off→on', async () => {
            const user = setupUser();
            server.use(
                hostAccessHandlers.userHostDetails.success(
                    createMockUserAccessDetail({
                        id: 5,
                        display_name: 'Charlie',
                        bypass_host_check: false,
                        devices: [createMockDeviceListItem({ id: 10, name: 'Charlie Laptop', live_address_count: 3 })],
                    }),
                ),
            );
            renderUserDetailPage(5);

            const bypassSwitch = await screen.findByRole('switch', { name: /bypass host check/i });
            await user.click(bypassSwitch);

            const saveButton = await screen.findByRole('button', { name: /save changes/i });
            expect(saveButton).toHaveAttribute('data-disabled', 'true');
            expect(screen.getByText(/enabling bypass lets charlie reach all hosts/i)).toBeInTheDocument();

            const ackCheckbox = screen.getByRole('checkbox', { name: /i understand this exposes/i });
            await user.click(ackCheckbox);

            expect(saveButton).not.toHaveAttribute('data-disabled', 'true');
        });

        it('does not show the warning bar for an already-bypassed user', async () => {
            const user = setupUser();
            server.use(
                hostAccessHandlers.userHostDetails.success(
                    createMockUserAccessDetail({
                        id: 5,
                        display_name: 'Charlie',
                        bypass_host_check: true,
                        devices: [createMockDeviceListItem({ id: 10, name: 'Charlie Laptop' })],
                    }),
                ),
            );
            renderUserDetailPage(5);

            const bypassSwitch = await screen.findByRole('switch', { name: /bypass host check/i });
            // Turn bypass off: a real edit to an already-bypassed user. Groups are inert
            // under bypass, so the switch is the only lever — and turning it off dirties
            // the draft without being an off→on enable, so the acknowledge warning must
            // not appear.
            await user.click(bypassSwitch);

            const saveButton = await screen.findByRole('button', { name: /save changes/i });
            expect(saveButton).not.toHaveAttribute('data-disabled', 'true');
            expect(screen.queryByText(/enabling bypass lets/i)).not.toBeInTheDocument();
        });
    });
});
