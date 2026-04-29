import { beforeEach, describe, expect, it, vi } from 'vitest';
import { screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { http, HttpResponse } from 'msw';
import { HostsPage } from '@/pages/HostsPage';
import { AuthProvider } from '@/features/auth/AuthContext';
import { TEST_TIMEOUTS } from '@/test/constants';
import { endpoints, hostAccessHandlers } from '@/test/mocks/handlers';
import { server } from '@/test/setup';
import { renderWithProviders } from '@/test/utils';
import {
    createMockHostGroupWithMembers,
    createMockHostSuggestion,
    createMockHostSuggestionsPage,
    createMockKnownHostWithStats,
} from '@/test/mocks/data';

// vi.mock is hoisted above imports; vi.hoisted makes the spy available to the factory.
const { mockGuard } = vi.hoisted(() => ({ mockGuard: vi.fn() }));
vi.mock('@/hooks/useUnsavedChangesGuard', () => ({
    useUnsavedChangesGuard: (isDirty: boolean) => mockGuard(isDirty),
}));

function renderHostsPage() {
    return renderWithProviders(<AuthProvider><HostsPage /></AuthProvider>);
}

describe('HostsPage', () => {
    beforeEach(() => mockGuard.mockClear());

    // ─── B1: Happy path ──────────────────────────────────────────────────────────

    describe('happy path', () => {
        it('renders the page heading and three tabs', () => {
            renderHostsPage();
            expect(screen.getByRole('heading', { name: 'Hosts', level: 1 })).toBeInTheDocument();
            expect(screen.getByRole('tab', { name: /known hosts/i })).toBeInTheDocument();
            expect(screen.getByRole('tab', { name: /groups/i })).toBeInTheDocument();
            expect(screen.getByRole('tab', { name: /suggestions/i })).toBeInTheDocument();
        });

        it('shows correct count badges in each tab after data loads', async () => {
            server.use(
                hostAccessHandlers.listKnownHosts.success([
                    createMockKnownHostWithStats({ id: 1, fqdn: 'app.lan' }),
                    createMockKnownHostWithStats({ id: 2, fqdn: 'db.lan' }),
                ]),
                hostAccessHandlers.listHostGroups.success([
                    createMockHostGroupWithMembers({ id: 1, name: 'Internal' }),
                ]),
                hostAccessHandlers.listHostSuggestions.success(
                    createMockHostSuggestionsPage({
                        suggestions: [createMockHostSuggestion({ fqdn: 'unknown.lan' })],
                    }),
                ),
            );
            renderHostsPage();

            await waitFor(() => {
                expect(within(screen.getByRole('tab', { name: /known hosts/i })).getByText('2')).toBeInTheDocument();
                expect(within(screen.getByRole('tab', { name: /groups/i })).getByText('1')).toBeInTheDocument();
                expect(within(screen.getByRole('tab', { name: /suggestions/i })).getByText('1')).toBeInTheDocument();
            }, { timeout: TEST_TIMEOUTS.MEDIUM });
        });

        it('Known hosts panel shows one row per server host', async () => {
            server.use(
                hostAccessHandlers.listKnownHosts.success([
                    createMockKnownHostWithStats({ id: 1, fqdn: 'app.lan' }),
                    createMockKnownHostWithStats({ id: 2, fqdn: 'db.lan' }),
                ]),
            );
            renderHostsPage();

            await waitFor(() => {
                expect(screen.getByText('app.lan')).toBeInTheDocument();
                expect(screen.getByText('db.lan')).toBeInTheDocument();
            }, { timeout: TEST_TIMEOUTS.MEDIUM });
        });
    });

    // ─── B3: Server → draft sync / initial clean state ───────────────────────────

    describe('initial clean state', () => {
        it('no StagedChangesBar is visible after a clean load', async () => {
            server.use(
                hostAccessHandlers.listKnownHosts.success([
                    createMockKnownHostWithStats({ id: 1, fqdn: 'host.lan' }),
                ]),
            );
            renderHostsPage();

            await waitFor(() => {
                expect(screen.getByText('host.lan')).toBeInTheDocument();
            }, { timeout: TEST_TIMEOUTS.MEDIUM });

            expect(screen.queryByRole('button', { name: /save changes/i })).not.toBeInTheDocument();
        });
    });

    // ─── B4: Cross-tab lock ──────────────────────────────────────────────────────

    describe('cross-tab lock', () => {
        async function stageOneSuggestion(fqdn: string) {
            server.use(
                hostAccessHandlers.listHostSuggestions.success(
                    createMockHostSuggestionsPage({ suggestions: [createMockHostSuggestion({ fqdn })] }),
                ),
            );
            renderHostsPage();
            await userEvent.click(screen.getByRole('tab', { name: /suggestions/i }));
            await waitFor(() => expect(screen.getByText(fqdn)).toBeInTheDocument(), { timeout: TEST_TIMEOUTS.MEDIUM });
            await userEvent.click(screen.getByRole('button', { name: /add to known/i }));
        }

        it('dirty hosts → Groups tab shows "Known hosts tab has unsaved changes" lock alert', async () => {
            await stageOneSuggestion('new.lan');

            await userEvent.click(screen.getByRole('tab', { name: /groups/i }));

            await waitFor(() => {
                expect(screen.getByText('Known hosts tab has unsaved changes')).toBeInTheDocument();
                expect(screen.getByRole('button', { name: /discard host changes/i })).toBeInTheDocument();
            }, { timeout: TEST_TIMEOUTS.SHORT });
        });

        it('clicking "Discard host changes" dismisses the lock alert', async () => {
            await stageOneSuggestion('new.lan');

            await userEvent.click(screen.getByRole('tab', { name: /groups/i }));
            await waitFor(() => {
                expect(screen.getByRole('button', { name: /discard host changes/i })).toBeInTheDocument();
            }, { timeout: TEST_TIMEOUTS.SHORT });

            await userEvent.click(screen.getByRole('button', { name: /discard host changes/i }));

            await waitFor(() => {
                expect(screen.queryByText('Known hosts tab has unsaved changes')).not.toBeInTheDocument();
            }, { timeout: TEST_TIMEOUTS.SHORT });
        });

        it('dirty groups → Known hosts tab shows "Groups tab has unsaved changes" lock alert', async () => {
            renderHostsPage();

            await userEvent.click(screen.getByRole('tab', { name: /groups/i }));
            await waitFor(() => {
                expect(screen.getByRole('button', { name: /new group/i })).toBeInTheDocument();
            }, { timeout: TEST_TIMEOUTS.MEDIUM });

            await userEvent.click(screen.getByRole('button', { name: /new group/i }));
            await waitFor(() => expect(screen.getByRole('dialog')).toBeInTheDocument());

            const dialog = screen.getByRole('dialog');
            await userEvent.type(within(dialog).getByRole('textbox', { name: /name/i }), 'TestGroup');
            await userEvent.click(within(dialog).getByRole('button', { name: /^create$/i }));

            await userEvent.click(screen.getByRole('tab', { name: /known hosts/i }));

            await waitFor(() => {
                expect(screen.getByText('Groups tab has unsaved changes')).toBeInTheDocument();
            }, { timeout: TEST_TIMEOUTS.SHORT });
        });
    });

    // ─── B5: Suggestions staging filter ─────────────────────────────────────────

    describe('suggestions staging', () => {
        it('staging a suggestion removes it from the list while leaving others', async () => {
            server.use(
                hostAccessHandlers.listHostSuggestions.success(
                    createMockHostSuggestionsPage({
                        suggestions: [
                            createMockHostSuggestion({ fqdn: 'foo.lan' }),
                            createMockHostSuggestion({ fqdn: 'bar.lan' }),
                        ],
                    }),
                ),
            );
            renderHostsPage();

            await userEvent.click(screen.getByRole('tab', { name: /suggestions/i }));
            await waitFor(() => {
                expect(screen.getByText('foo.lan')).toBeInTheDocument();
                expect(screen.getByText('bar.lan')).toBeInTheDocument();
            }, { timeout: TEST_TIMEOUTS.MEDIUM });

            const fooRow = screen.getAllByRole('row').find((row) => within(row).queryByText('foo.lan'));
            await userEvent.click(within(fooRow!).getByRole('button', { name: /add to known/i }));

            await waitFor(() => {
                expect(screen.queryByText('foo.lan')).not.toBeInTheDocument();
                expect(screen.getByText('bar.lan')).toBeInTheDocument();
            }, { timeout: TEST_TIMEOUTS.SHORT });
        });

        it('staging a suggestion decrements the Suggestions badge and shows the StagedChangesBar', async () => {
            server.use(
                hostAccessHandlers.listHostSuggestions.success(
                    createMockHostSuggestionsPage({
                        suggestions: [
                            createMockHostSuggestion({ fqdn: 'foo.lan' }),
                            createMockHostSuggestion({ fqdn: 'bar.lan' }),
                        ],
                    }),
                ),
            );
            renderHostsPage();

            await userEvent.click(screen.getByRole('tab', { name: /suggestions/i }));
            await waitFor(() => {
                expect(within(screen.getByRole('tab', { name: /suggestions/i })).getByText('2')).toBeInTheDocument();
            }, { timeout: TEST_TIMEOUTS.MEDIUM });

            const fooRow = screen.getAllByRole('row').find((row) => within(row).queryByText('foo.lan'));
            await userEvent.click(within(fooRow!).getByRole('button', { name: /add to known/i }));

            // Suggestions count drops to 1; StagedChangesBar becomes visible (dirty state)
            await waitFor(() => {
                expect(within(screen.getByRole('tab', { name: /suggestions/i })).getByText('1')).toBeInTheDocument();
            }, { timeout: TEST_TIMEOUTS.SHORT });
            // Switch to Known hosts to see the bar (keepMounted=false means it only renders on active tab)
            await userEvent.click(screen.getByRole('tab', { name: /known hosts/i }));
            await waitFor(() => {
                expect(screen.getByRole('button', { name: /save changes/i })).toBeInTheDocument();
            }, { timeout: TEST_TIMEOUTS.MEDIUM });
        });
    });

    // ─── B6: Error state ─────────────────────────────────────────────────────────

    describe('error state', () => {
        it('shows error message when the suggestions query fails', async () => {
            server.use(hostAccessHandlers.listHostSuggestions.serverError());
            renderHostsPage();

            await userEvent.click(screen.getByRole('tab', { name: /suggestions/i }));

            await waitFor(() => {
                expect(screen.getByText('Failed to load suggestions.')).toBeInTheDocument();
            }, { timeout: TEST_TIMEOUTS.MEDIUM });
        });
    });

    // ─── B7: Save flow smoke test ────────────────────────────────────────────────
    // Stages a host via AddHostModal (stays on Known hosts tab throughout to avoid
    // the tab-switch/fetchQuery/unmount race). The Suggestions→navigate→save path
    // is not covered here; that integration is exercised manually.
    //
    // Uses structuralSharing:false on the QueryClient so the post-save refetch always
    // returns a new array reference. Without it, React Query's structural sharing keeps
    // the same [] reference, the draft-sync useEffect never fires, and the bar stays dirty.

    describe('save flow', () => {
        it('reconcile is called and the StagedChangesBar disappears after a successful save', async () => {
            const reconcileSpy = vi.fn();
            server.use(
                http.put(endpoints.adminHostsReconcile, () => {
                    reconcileSpy();
                    return new HttpResponse(null, { status: 204 });
                }),
            );

            const { QueryClient } = await import('@tanstack/react-query');
            const queryClient = new QueryClient({
                defaultOptions: {
                    queries: { retry: false, structuralSharing: false },
                    mutations: { retry: false },
                },
            });
            renderWithProviders(<AuthProvider><HostsPage /></AuthProvider>, { queryClient });

            // Known hosts tab is default; wait for the empty-state "Add host" button
            await waitFor(() => {
                expect(screen.getByRole('button', { name: /add host/i })).toBeInTheDocument();
            }, { timeout: TEST_TIMEOUTS.MEDIUM });

            await userEvent.click(screen.getByRole('button', { name: /add host/i }));
            await waitFor(() => expect(screen.getByRole('dialog')).toBeInTheDocument());

            const dialog = screen.getByRole('dialog');
            await userEvent.type(within(dialog).getByRole('textbox', { name: /fqdn/i }), 'new.lan');
            await userEvent.click(within(dialog).getByRole('button', { name: /Add to draft/i }));

            await waitFor(() => {
                expect(screen.getByRole('button', { name: /save changes/i })).toBeInTheDocument();
            }, { timeout: TEST_TIMEOUTS.SHORT });

            await userEvent.click(screen.getByRole('button', { name: /save changes/i }));

            await waitFor(() => expect(reconcileSpy).toHaveBeenCalledOnce(), { timeout: TEST_TIMEOUTS.LONG });
            await waitFor(() => {
                expect(screen.queryByRole('button', { name: /save changes/i })).not.toBeInTheDocument();
            }, { timeout: TEST_TIMEOUTS.MEDIUM });
        });
    });

    // ─── B8: useUnsavedChangesGuard hookup ──────────────────────────────────────

    describe('useUnsavedChangesGuard hookup', () => {
        it('is called with false initially and with true after staging a host', async () => {
            server.use(
                hostAccessHandlers.listHostSuggestions.success(
                    createMockHostSuggestionsPage({
                        suggestions: [createMockHostSuggestion({ fqdn: 'guard-test.lan' })],
                    }),
                ),
            );
            renderHostsPage();

            await waitFor(() => {
                expect(mockGuard).toHaveBeenCalledWith(false);
            }, { timeout: TEST_TIMEOUTS.SHORT });

            await userEvent.click(screen.getByRole('tab', { name: /suggestions/i }));
            await waitFor(() => expect(screen.getByText('guard-test.lan')).toBeInTheDocument(), { timeout: TEST_TIMEOUTS.MEDIUM });
            await userEvent.click(screen.getByRole('button', { name: /add to known/i }));

            await waitFor(() => {
                expect(mockGuard).toHaveBeenCalledWith(true);
            }, { timeout: TEST_TIMEOUTS.SHORT });
        });
    });
});
