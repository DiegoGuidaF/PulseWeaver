import { beforeEach, describe, expect, it } from 'vitest';
import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Route, Routes } from 'react-router-dom';
import { delay, http } from 'msw';
import { UsersPage } from '@/pages/access/users/UsersPage';
import { AuthProvider } from '@/features/auth/AuthContext';
import { TEST_TIMEOUTS } from '@/test/constants';
import { authHandlers, endpoints, hostAccessHandlers, responses } from '@/test/mocks/handlers';
import { server } from '@/test/setup';
import { renderWithProviders } from '@/test/utils';
import { createMockUser, createMockUserListItem } from '@/test/mocks/data';
import { UserRole } from '@/lib/api';

function renderUsersPage() {
    return renderWithProviders(<AuthProvider><UsersPage /></AuthProvider>);
}

describe('UsersPage', () => {
    describe('basic rendering', () => {
        beforeEach(() => {
            const user1 = createMockUser({ id: 1, username: 'alice', display_name: 'Alice', role: UserRole.USER });
            const user2 = createMockUser({ id: 2, username: 'bob', display_name: 'Bob', role: UserRole.ADMIN });
            const summary1 = createMockUserListItem({ id: 1, username: 'alice', display_name: 'Alice', role: UserRole.USER, bypass_host_check: false, host_count: 0 });
            const summary2 = createMockUserListItem({ id: 2, username: 'bob', display_name: 'Bob', role: UserRole.ADMIN, bypass_host_check: false, host_count: 0 });

            server.use(
                authHandlers.me.success({ id: 1, username: 'alice', display_name: 'Alice', role: UserRole.USER }),
                authHandlers.listUsers.success([user1, user2]),
                hostAccessHandlers.listUsersHostAccess.success([summary1, summary2]),
            );
        });

        it('renders heading, New user button, and one row per returned user', async () => {
            renderUsersPage();

            await waitFor(
                () => {
                    expect(screen.getByRole('heading', { name: 'Users', level: 1 })).toBeInTheDocument();
                    expect(screen.getByRole('button', { name: /new user/i })).toBeInTheDocument();
                    expect(screen.getByText('alice')).toBeInTheDocument();
                    expect(screen.getByText('bob')).toBeInTheDocument();
                },
                { timeout: TEST_TIMEOUTS.SHORT },
            );
        });

        it('hides action buttons for the current authenticated user\'s own row', async () => {
            renderUsersPage();

            await waitFor(
                () => {
                    expect(screen.getByText('alice')).toBeInTheDocument();
                    // bob (admin, not self) gets a Demote button
                    expect(screen.getByRole('button', { name: 'Demote to user' })).toBeInTheDocument();
                },
                { timeout: TEST_TIMEOUTS.MEDIUM },
            );

            // alice is the authenticated user — no promote/delete buttons on her row
            expect(screen.queryByRole('button', { name: 'Promote to admin' })).not.toBeInTheDocument();
            expect(screen.queryByRole('button', { name: 'Delete alice' })).not.toBeInTheDocument();
        });
    });

    describe('host-access cell variants', () => {
        it('renders a count badge when user has hosts (host_count: 3, bypass_host_check: false)', async () => {
            const user = createMockUser({ id: 5, username: 'charlie', display_name: 'Charlie', role: UserRole.USER });
            const summary = createMockUserListItem({ id: 5, username: 'charlie', display_name: 'Charlie', role: UserRole.USER, bypass_host_check: false, host_count: 3 });

            server.use(
                authHandlers.me.success({ id: 99, username: 'admin', role: UserRole.ADMIN }),
                authHandlers.listUsers.success([user]),
                hostAccessHandlers.listUsersHostAccess.success([summary]),
            );

            renderUsersPage();

            await waitFor(
                () => {
                    expect(screen.getByText('3 hosts')).toBeInTheDocument();
                },
                { timeout: TEST_TIMEOUTS.SHORT },
            );
        });

        it('renders "bypass ✱" badge when bypass_host_check: true', async () => {
            const user = createMockUser({ id: 5, username: 'charlie', display_name: 'Charlie', role: UserRole.USER });
            const summary = createMockUserListItem({ id: 5, username: 'charlie', display_name: 'Charlie', role: UserRole.USER, bypass_host_check: true, host_count: 0 });

            server.use(
                authHandlers.me.success({ id: 99, username: 'admin', role: UserRole.ADMIN }),
                authHandlers.listUsers.success([user]),
                hostAccessHandlers.listUsersHostAccess.success([summary]),
            );

            renderUsersPage();

            await waitFor(
                () => {
                    expect(screen.getByText('bypass ✱')).toBeInTheDocument();
                },
                { timeout: TEST_TIMEOUTS.SHORT },
            );
        });

        it('renders em-dash when host_count: 0 and bypass_host_check: false', async () => {
            const user = createMockUser({ id: 5, username: 'charlie', display_name: 'Charlie', role: UserRole.USER });
            const summary = createMockUserListItem({ id: 5, username: 'charlie', display_name: 'Charlie', role: UserRole.USER, bypass_host_check: false, host_count: 0 });

            server.use(
                authHandlers.me.success({ id: 99, username: 'admin', role: UserRole.ADMIN }),
                authHandlers.listUsers.success([user]),
                hostAccessHandlers.listUsersHostAccess.success([summary]),
            );

            renderUsersPage();

            await waitFor(
                () => {
                    expect(screen.getByText('charlie')).toBeInTheDocument();
                },
                { timeout: TEST_TIMEOUTS.SHORT },
            );

            const dashElements = screen.getAllByText('—');
            expect(dashElements.length).toBeGreaterThan(0);
        });
    });

    describe('promote button opens role modal', () => {
        it('clicking the promote button opens the role confirmation modal', async () => {
            const user = createMockUser({ id: 5, username: 'charlie', display_name: 'Charlie', role: UserRole.USER });
            const summary = createMockUserListItem({ id: 5, username: 'charlie', display_name: 'Charlie', role: UserRole.USER, bypass_host_check: false, host_count: 1 });

            server.use(
                authHandlers.me.success({ id: 99, username: 'admin', role: UserRole.ADMIN }),
                authHandlers.listUsers.success([user]),
                hostAccessHandlers.listUsersHostAccess.success([summary]),
            );

            renderUsersPage();

            await waitFor(
                () => {
                    expect(screen.getByRole('button', { name: 'Promote to admin' })).toBeInTheDocument();
                },
                { timeout: TEST_TIMEOUTS.SHORT },
            );

            await userEvent.click(screen.getByRole('button', { name: 'Promote to admin' }));

            await waitFor(
                () => {
                    expect(screen.getByRole('dialog')).toBeInTheDocument();
                },
                { timeout: TEST_TIMEOUTS.SHORT },
            );
        });
    });

    describe('action buttons', () => {
        it('shows "Promote to admin" button for a user-role row', async () => {
            const user = createMockUser({ id: 5, username: 'charlie', display_name: 'Charlie', role: UserRole.USER });
            const summary = createMockUserListItem({ id: 5, username: 'charlie', display_name: 'Charlie', role: UserRole.USER });

            server.use(
                authHandlers.me.success({ id: 99, username: 'admin', role: UserRole.ADMIN }),
                authHandlers.listUsers.success([user]),
                hostAccessHandlers.listUsersHostAccess.success([summary]),
            );

            renderUsersPage();

            await waitFor(
                () => {
                    expect(screen.getByRole('button', { name: 'Promote to admin' })).toBeInTheDocument();
                },
                { timeout: TEST_TIMEOUTS.SHORT },
            );
        });

        it('shows "Demote to user" button for an admin-role row', async () => {
            const user = createMockUser({ id: 5, username: 'charlie', display_name: 'Charlie', role: UserRole.ADMIN });
            const summary = createMockUserListItem({ id: 5, username: 'charlie', display_name: 'Charlie', role: UserRole.ADMIN });

            server.use(
                authHandlers.me.success({ id: 99, username: 'admin', role: UserRole.ADMIN }),
                authHandlers.listUsers.success([user]),
                hostAccessHandlers.listUsersHostAccess.success([summary]),
            );

            renderUsersPage();

            await waitFor(
                () => {
                    expect(screen.getByRole('button', { name: 'Demote to user' })).toBeInTheDocument();
                },
                { timeout: TEST_TIMEOUTS.SHORT },
            );
        });

        it('does NOT render action buttons for the current user\'s own row', async () => {
            const user = createMockUser({ id: 1, username: 'alice', display_name: 'Alice', role: UserRole.ADMIN });
            const summary = createMockUserListItem({ id: 1, username: 'alice', display_name: 'Alice', role: UserRole.ADMIN });

            server.use(
                authHandlers.me.success({ id: 1, username: 'alice', role: UserRole.ADMIN }),
                authHandlers.listUsers.success([user]),
                hostAccessHandlers.listUsersHostAccess.success([summary]),
            );

            renderUsersPage();

            await waitFor(
                () => {
                    expect(screen.getByText('alice')).toBeInTheDocument();
                },
                { timeout: TEST_TIMEOUTS.SHORT },
            );

            expect(screen.queryByRole('button', { name: 'Demote to user' })).not.toBeInTheDocument();
            expect(screen.queryByRole('button', { name: 'Delete alice' })).not.toBeInTheDocument();
        });

        it('does NOT render action buttons for a superadmin row', async () => {
            const user = createMockUser({ id: 5, username: 'sa_user', display_name: 'Super Admin', role: UserRole.SUPERADMIN });
            const summary = createMockUserListItem({ id: 5, username: 'sa_user', display_name: 'Super Admin', role: UserRole.SUPERADMIN });

            server.use(
                authHandlers.me.success({ id: 99, username: 'admin', role: UserRole.ADMIN }),
                authHandlers.listUsers.success([user]),
                hostAccessHandlers.listUsersHostAccess.success([summary]),
            );

            renderUsersPage();

            await waitFor(
                () => {
                    expect(screen.getByText('sa_user')).toBeInTheDocument();
                },
                { timeout: TEST_TIMEOUTS.SHORT },
            );

            expect(screen.queryByRole('button', { name: 'Promote to admin' })).not.toBeInTheDocument();
            expect(screen.queryByRole('button', { name: 'Delete sa_user' })).not.toBeInTheDocument();
        });

        it('clicking the delete button opens the delete confirmation modal', async () => {
            const user = createMockUser({ id: 5, username: 'charlie', display_name: 'Charlie', role: UserRole.USER });
            const summary = createMockUserListItem({ id: 5, username: 'charlie', display_name: 'Charlie', role: UserRole.USER });

            server.use(
                authHandlers.me.success({ id: 99, username: 'admin', role: UserRole.ADMIN }),
                authHandlers.listUsers.success([user]),
                hostAccessHandlers.listUsersHostAccess.success([summary]),
            );

            renderUsersPage();

            await waitFor(
                () => {
                    expect(screen.getByRole('button', { name: 'Delete charlie' })).toBeInTheDocument();
                },
                { timeout: TEST_TIMEOUTS.SHORT },
            );

            await userEvent.click(screen.getByRole('button', { name: 'Delete charlie' }));

            await waitFor(
                () => {
                    expect(screen.getByRole('dialog')).toBeInTheDocument();
                },
                { timeout: TEST_TIMEOUTS.SHORT },
            );
        });
    });

    describe('loading and error states', () => {
        it('shows a loader and hides the table while data is loading', () => {
            server.use(
                http.get(endpoints.usersHostAccess, async () => {
                    await delay('infinite');
                    return responses.ok([]);
                }),
            );

            renderUsersPage();

            expect(screen.queryByRole('heading', { name: 'Users', level: 1 })).not.toBeInTheDocument();
        });

        it('shows an error message when the user list fails to load', async () => {
            server.use(hostAccessHandlers.listUsersHostAccess.serverError());

            renderUsersPage();

            await waitFor(
                () => {
                    expect(screen.getByText('Failed to load users')).toBeInTheDocument();
                    expect(screen.getByRole('button', { name: /try again/i })).toBeInTheDocument();
                },
                { timeout: TEST_TIMEOUTS.SHORT },
            );
        });
    });

    describe('create user modal', () => {
        it('clicking "+ New user" opens the create user modal', async () => {
            const user = userEvent.setup();

            server.use(
                authHandlers.me.success({ id: 99, username: 'admin', role: UserRole.ADMIN }),
                hostAccessHandlers.listUsersHostAccess.success([]),
            );

            renderUsersPage();

            await waitFor(
                () => {
                    expect(screen.getByRole('button', { name: /new user/i })).toBeInTheDocument();
                },
                { timeout: TEST_TIMEOUTS.SHORT },
            );

            await user.click(screen.getByRole('button', { name: /new user/i }));

            await waitFor(
                () => {
                    expect(screen.getByRole('dialog')).toBeInTheDocument();
                },
                { timeout: TEST_TIMEOUTS.SHORT },
            );
        });
    });

    describe('row click navigation', () => {
        function renderWithRoutes() {
            function TestApp() {
                return (
                    <Routes>
                        <Route path="/" element={<AuthProvider><UsersPage /></AuthProvider>} />
                        <Route path="/access/users/:id" element={<div data-testid="user-detail" />} />
                    </Routes>
                );
            }
            return renderWithProviders(<TestApp />);
        }

        it('clicking a non-superadmin row navigates to the user detail page', async () => {
            const user = userEvent.setup();
            const summary = createMockUserListItem({ id: 5, username: 'charlie', display_name: 'Charlie', role: UserRole.USER });

            server.use(
                authHandlers.me.success({ id: 99, username: 'admin', role: UserRole.ADMIN }),
                hostAccessHandlers.listUsersHostAccess.success([summary]),
            );

            renderWithRoutes();

            await waitFor(
                () => expect(screen.getByText('charlie')).toBeInTheDocument(),
                { timeout: TEST_TIMEOUTS.SHORT },
            );

            await user.click(screen.getByText('charlie'));

            await waitFor(
                () => expect(screen.getByTestId('user-detail')).toBeInTheDocument(),
                { timeout: TEST_TIMEOUTS.SHORT },
            );
        });

        it('clicking a superadmin row does not navigate', async () => {
            const user = userEvent.setup();
            const summary = createMockUserListItem({ id: 5, username: 'sa_user', display_name: 'Super Admin', role: UserRole.SUPERADMIN });

            server.use(
                authHandlers.me.success({ id: 99, username: 'admin', role: UserRole.ADMIN }),
                hostAccessHandlers.listUsersHostAccess.success([summary]),
            );

            renderWithRoutes();

            await waitFor(
                () => expect(screen.getByText('sa_user')).toBeInTheDocument(),
                { timeout: TEST_TIMEOUTS.SHORT },
            );

            await user.click(screen.getByText('sa_user'));

            expect(screen.queryByTestId('user-detail')).not.toBeInTheDocument();
            expect(screen.getByRole('heading', { name: 'Users', level: 1 })).toBeInTheDocument();
        });
    });

    describe('group_id URL param', () => {
        it('pre-seeds the group filter, showing only users in that group', async () => {
            const inGroup = createMockUserListItem({
                id: 1,
                username: 'alice',
                display_name: 'Alice',
                groups: [{ id: 5, name: 'Engineering' }],
            });
            const notInGroup = createMockUserListItem({
                id: 2,
                username: 'bob',
                display_name: 'Bob',
                groups: [],
            });

            server.use(
                authHandlers.me.success({ id: 99, username: 'admin', role: UserRole.ADMIN }),
                hostAccessHandlers.listUsersHostAccess.success([inGroup, notInGroup]),
            );

            renderWithProviders(
                <AuthProvider><UsersPage /></AuthProvider>,
                { initialEntries: ['/access/users?group_id=5'] },
            );

            await waitFor(
                () => expect(screen.getByText('alice')).toBeInTheDocument(),
                { timeout: TEST_TIMEOUTS.SHORT },
            );

            expect(screen.queryByText('bob')).not.toBeInTheDocument();
        });
    });
});
