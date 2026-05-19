import { describe, expect, it, vi } from 'vitest';
import { screen, waitFor } from '@testing-library/react';
import { HostGroupsPage } from '@/pages/access/host-groups/HostGroupsPage';
import { server } from '@/test/setup';
import { renderWithProviders } from '@/test/utils';
import { TEST_TIMEOUTS } from '@/test/constants';
import { hostAccessHandlers } from '@/test/mocks/handlers';
import { createMockGroupDetailWithUsers } from '@/test/mocks/data';

const { mockGuard } = vi.hoisted(() => ({ mockGuard: vi.fn() }));
vi.mock('@/hooks/useUnsavedChangesGuard', () => ({
    useUnsavedChangesGuard: (isDirty: boolean) => mockGuard(isDirty),
}));

function renderHostGroupsPage() {
    return renderWithProviders(<HostGroupsPage />);
}

describe('HostGroupsPage', () => {
    // ─── Happy path ──────────────────────────────────────────────────────────────

    describe('happy path', () => {
        it('renders the page heading', () => {
            renderHostGroupsPage();
            expect(screen.getByRole('heading', { name: 'Host Groups', level: 1 })).toBeInTheDocument();
        });

        it('renders group names after data loads', async () => {
            server.use(
                hostAccessHandlers.listHostGroups.success([
                    createMockGroupDetailWithUsers({ id: 1, name: 'Engineering' }),
                    createMockGroupDetailWithUsers({ id: 2, name: 'Marketing' }),
                ]),
            );
            renderHostGroupsPage();

            // First group is auto-selected; its name appears in both the master list and detail panel
            await waitFor(() => {
                expect(screen.getAllByText('Engineering').length).toBeGreaterThanOrEqual(1);
                expect(screen.getByText('Marketing')).toBeInTheDocument();
            }, { timeout: TEST_TIMEOUTS.MEDIUM });
        });
    });

    // ─── Unsaved changes guard ────────────────────────────────────────────────────

    describe('useUnsavedChangesGuard hookup', () => {
        it('is called with false initially (no staged changes)', async () => {
            renderHostGroupsPage();

            await waitFor(() => {
                expect(mockGuard).toHaveBeenCalledWith(false);
            }, { timeout: TEST_TIMEOUTS.SHORT });
        });
    });

    // ─── StagedChangesBar ─────────────────────────────────────────────────────────

    describe('StagedChangesBar', () => {
        it('is not visible after a clean load', async () => {
            server.use(
                hostAccessHandlers.listHostGroups.success([
                    createMockGroupDetailWithUsers({ id: 1, name: 'Dev Group' }),
                ]),
            );
            renderHostGroupsPage();

            // "Dev Group" appears in both master list and detail panel; getAllByText is safe here
            await waitFor(() => {
                expect(screen.getAllByText('Dev Group').length).toBeGreaterThanOrEqual(1);
            }, { timeout: TEST_TIMEOUTS.MEDIUM });

            expect(screen.queryByRole('button', { name: /save changes/i })).not.toBeInTheDocument();
        });
    });
});
