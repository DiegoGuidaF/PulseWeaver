import { describe, expect, it, vi } from 'vitest';
import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { NetworkPolicyDetailPage } from '@/pages/access/network-policies/NetworkPolicyDetailPage';
import { server } from '@/test/setup';
import { renderWithProviders } from '@/test/utils';
import { TEST_TIMEOUTS } from '@/test/constants';
import { networkPolicyHandlers } from '@/test/mocks/handlers';
import { createMockSubjectGroupDetail } from '@/test/mocks/data';

const { mockGuard } = vi.hoisted(() => ({ mockGuard: vi.fn() }));
vi.mock('@/hooks/useUnsavedChangesGuard', () => ({
    useUnsavedChangesGuard: (isDirty: boolean) => mockGuard(isDirty),
}));

function renderPage(id = '1') {
    return renderWithProviders(<NetworkPolicyDetailPage />, {
        initialEntries: [`/access/network-policies/${id}`],
        path: '/access/network-policies/:id',
    });
}

describe('NetworkPolicyDetailPage', () => {
    // ─── Happy path ──────────────────────────────────────────────────────────────

    describe('happy path', () => {
        it('renders policy name and CIDR after data loads', async () => {
            server.use(
                networkPolicyHandlers.get.success({
                    name: 'Office Network',
                    cidr: '192.168.1.0/24',
                }),
            );
            renderPage();

            await waitFor(() => {
                expect(screen.getByText('Office Network')).toBeInTheDocument();
                expect(screen.getByText('192.168.1.0/24')).toBeInTheDocument();
            }, { timeout: TEST_TIMEOUTS.MEDIUM });
        });

        it('shows Enabled badge when policy is enabled', async () => {
            server.use(networkPolicyHandlers.get.success({ enabled: true }));
            renderPage();

            await waitFor(() => {
                expect(screen.getByText('Enabled')).toBeInTheDocument();
            }, { timeout: TEST_TIMEOUTS.MEDIUM });
        });

        it('shows Disabled badge when policy is disabled', async () => {
            server.use(networkPolicyHandlers.get.success({ enabled: false }));
            renderPage();

            await waitFor(() => {
                expect(screen.getByText('Disabled')).toBeInTheDocument();
            }, { timeout: TEST_TIMEOUTS.MEDIUM });
        });

        it('renders group names in the subject groups panel', async () => {
            server.use(
                networkPolicyHandlers.get.success({
                    groups: [
                        // granted: false keeps group names out of EffectiveHostsPanel filter badges
                        createMockSubjectGroupDetail({ id: 1, name: 'Engineering', granted: false }),
                        createMockSubjectGroupDetail({ id: 2, name: 'Marketing', granted: false }),
                    ],
                }),
            );
            renderPage();

            await waitFor(() => {
                expect(screen.getByText('Engineering')).toBeInTheDocument();
                expect(screen.getByText('Marketing')).toBeInTheDocument();
            }, { timeout: TEST_TIMEOUTS.MEDIUM });
        });
    });

    // ─── Error state ─────────────────────────────────────────────────────────────

    describe('error state', () => {
        it('shows not-found alert when the policy fetch fails', async () => {
            server.use(networkPolicyHandlers.get.notFound());
            renderPage();

            await waitFor(() => {
                expect(screen.getByText('Could not load policy')).toBeInTheDocument();
                expect(screen.getByRole('button', { name: /try again/i })).toBeInTheDocument();
            }, { timeout: TEST_TIMEOUTS.MEDIUM });
        });
    });

    // ─── Delete flow ─────────────────────────────────────────────────────────────

    describe('delete flow', () => {
        it('navigates away after a successful delete', async () => {
            const user = userEvent.setup();
            server.use(
                networkPolicyHandlers.get.success({ name: 'To Delete' }),
            );
            renderPage();

            await waitFor(() => {
                expect(screen.getByText('To Delete')).toBeInTheDocument();
            }, { timeout: TEST_TIMEOUTS.MEDIUM });

            // open the ⋯ menu
            await user.click(screen.getByRole('button', { name: '' }));
            await user.click(await screen.findByText('Delete policy'));

            // confirm in the modal
            const dialog = await screen.findByRole('dialog');
            await user.click(
                dialog
                    ? screen.getAllByRole('button', { name: /delete/i }).find(
                          (b) => b.closest('[role="dialog"]'),
                      )!
                    : screen.getByRole('button', { name: /delete/i }),
            );

            // after delete the route resolves to null — the page unmounts
            await waitFor(() => {
                expect(screen.queryByText('To Delete')).not.toBeInTheDocument();
            }, { timeout: TEST_TIMEOUTS.LONG });
        });
    });

    // ─── Unsaved changes guard ────────────────────────────────────────────────────

    describe('useUnsavedChangesGuard hookup', () => {
        it('is called with false when no draft changes have been made', async () => {
            renderPage();

            await waitFor(() => {
                expect(mockGuard).toHaveBeenCalledWith(false);
            }, { timeout: TEST_TIMEOUTS.MEDIUM });
        });
    });

    // ─── StagedChangesBar visibility ──────────────────────────────────────────────

    describe('StagedChangesBar', () => {
        it('is not visible after a clean load', async () => {
            renderPage();

            await waitFor(() => {
                expect(screen.getByText('Test Policy')).toBeInTheDocument();
            }, { timeout: TEST_TIMEOUTS.MEDIUM });

            expect(screen.queryByRole('button', { name: /save/i })).not.toBeInTheDocument();
        });

        it('appears when a group assignment is toggled', async () => {
            const user = userEvent.setup();
            server.use(
                networkPolicyHandlers.get.success({
                    groups: [
                        createMockSubjectGroupDetail({ id: 1, name: 'Dev Team', granted: false }),
                    ],
                }),
            );
            renderPage();

            await waitFor(() => {
                expect(screen.getByText('Dev Team')).toBeInTheDocument();
            }, { timeout: TEST_TIMEOUTS.MEDIUM });

            // The group checkbox has no accessible label; find it by role among all checkboxes
            const [groupCheckbox] = screen.getAllByRole('checkbox');
            await user.click(groupCheckbox);

            await waitFor(() => {
                expect(screen.getByRole('button', { name: /save/i })).toBeInTheDocument();
            }, { timeout: TEST_TIMEOUTS.SHORT });
        });
    });
});
