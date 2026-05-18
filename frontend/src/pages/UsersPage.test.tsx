import { beforeEach, describe, expect, it } from 'vitest';
import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { UsersPage } from '@/pages/UsersPage';
import { AuthProvider } from '@/features/auth/AuthContext';
import { TEST_TIMEOUTS } from '@/test/constants';
import { authHandlers, hostAccessHandlers } from '@/test/mocks/handlers';
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

        it('renders heading, Create user button, and one row per returned user', async () => {
            renderUsersPage();

            await waitFor(
                () => {
                    expect(screen.getByRole('heading', { name: 'Users', level: 1 })).toBeInTheDocument();
                    expect(screen.getByRole('button', { name: /create user/i })).toBeInTheDocument();
                    expect(screen.getByText('alice')).toBeInTheDocument();
                    expect(screen.getByText('bob')).toBeInTheDocument();
                },
                { timeout: TEST_TIMEOUTS.SHORT },
            );
        });

        it('shows "(you)" suffix on the current authenticated user\'s row', async () => {
            renderUsersPage();

            await waitFor(
                () => {
                    expect(screen.getByText('alice')).toBeInTheDocument();
                    expect(screen.getByText('(you)')).toBeInTheDocument();
                },
                { timeout: TEST_TIMEOUTS.MEDIUM },
            );
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

        it('renders "All hosts allowed" badge when bypass_host_check: true', async () => {
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
                    expect(screen.getByText('All hosts allowed')).toBeInTheDocument();
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

    describe('edit pencil opens drawer', () => {
        it('clicking the edit pencil opens the drawer titled with the user\'s display name', async () => {
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
                    expect(screen.getByRole('button', { name: 'Edit host access for charlie' })).toBeInTheDocument();
                },
                { timeout: TEST_TIMEOUTS.SHORT },
            );

            await userEvent.click(screen.getByRole('button', { name: 'Edit host access for charlie' }));

            await waitFor(
                () => {
                    expect(screen.getByText('Charlie')).toBeInTheDocument();
                },
                { timeout: TEST_TIMEOUTS.SHORT },
            );
        });
    });

    describe('kebab menu', () => {
        it('shows "Promote to admin" item for a user-role row', async () => {
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
                    expect(screen.getByRole('button', { name: 'More actions for charlie' })).toBeInTheDocument();
                },
                { timeout: TEST_TIMEOUTS.SHORT },
            );

            await userEvent.click(screen.getByRole('button', { name: 'More actions for charlie' }));

            await waitFor(
                () => {
                    expect(screen.getByText('Promote to admin')).toBeInTheDocument();
                },
                { timeout: TEST_TIMEOUTS.SHORT },
            );
        });

        it('shows "Demote to user" for an admin-role row', async () => {
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
                    expect(screen.getByRole('button', { name: 'More actions for charlie' })).toBeInTheDocument();
                },
                { timeout: TEST_TIMEOUTS.SHORT },
            );

            await userEvent.click(screen.getByRole('button', { name: 'More actions for charlie' }));

            await waitFor(
                () => {
                    expect(screen.getByText('Demote to user')).toBeInTheDocument();
                },
                { timeout: TEST_TIMEOUTS.SHORT },
            );
        });

        it('does NOT render kebab for the current user\'s own row', async () => {
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

            expect(screen.queryByRole('button', { name: 'More actions for alice' })).not.toBeInTheDocument();
        });

        it('does NOT render kebab for a superadmin row', async () => {
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

            expect(screen.queryByRole('button', { name: 'More actions for sa_user' })).not.toBeInTheDocument();
        });

        it('clicking "Delete user" inside the kebab opens the delete confirmation modal', async () => {
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
                    expect(screen.getByRole('button', { name: 'More actions for charlie' })).toBeInTheDocument();
                },
                { timeout: TEST_TIMEOUTS.SHORT },
            );

            await userEvent.click(screen.getByRole('button', { name: 'More actions for charlie' }));

            await waitFor(
                () => {
                    expect(screen.getByText('Delete user')).toBeInTheDocument();
                },
                { timeout: TEST_TIMEOUTS.SHORT },
            );

            await userEvent.click(screen.getByText('Delete user'));

            await waitFor(
                () => {
                    expect(screen.getByRole('dialog')).toBeInTheDocument();
                },
                { timeout: TEST_TIMEOUTS.SHORT },
            );
        });
    });
});
