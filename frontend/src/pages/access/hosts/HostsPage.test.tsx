import { beforeEach, describe, expect, it, vi } from 'vitest';
import { screen, waitFor, within } from '@testing-library/react';
import { http, HttpResponse } from 'msw';
import { HostsPage } from '@/pages/access/hosts/HostsPage';
import { AuthProvider } from '@/features/auth/AuthContext';
import { TEST_TIMEOUTS } from '@/test/constants';
import { endpoints, hostAccessHandlers } from '@/test/mocks/handlers';
import { server } from '@/test/setup';
import { renderWithProviders, setupUser } from '@/test/utils';
import {
    createMockHost,
    createMockHostSuggestion,
    createMockHostSuggestionsPage,
} from '@/test/mocks/data';

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
        it('renders the page heading, the known-hosts area and the suggestions section together', () => {
            renderHostsPage();
            expect(screen.getByRole('heading', { name: 'Hosts', level: 1 })).toBeInTheDocument();
            expect(
                screen.getByRole('button', { name: /observed in recent traffic/i }),
            ).toBeInTheDocument();
        });

        it('shows known hosts and the suggestions count badge together after data loads', async () => {
            server.use(
                hostAccessHandlers.listKnownHosts.success([
                    createMockHost({ id: 1, fqdn: 'app.lan' }),
                    createMockHost({ id: 2, fqdn: 'db.lan' }),
                ]),
                hostAccessHandlers.listHostSuggestions.success(
                    createMockHostSuggestionsPage({
                        suggestions: [createMockHostSuggestion({ fqdn: 'unknown.lan' })],
                    }),
                ),
            );
            renderHostsPage();

            await waitFor(() => {
                expect(screen.getByText('app.lan')).toBeInTheDocument();
                expect(screen.getByText('db.lan')).toBeInTheDocument();
                expect(screen.getByText('unknown.lan')).toBeInTheDocument();
                const suggestionsToggle = screen.getByRole('button', { name: /observed in recent traffic/i });
                expect(within(suggestionsToggle).getByText('1')).toBeInTheDocument();
            }, { timeout: TEST_TIMEOUTS.MEDIUM });
        });

        it('Hosts panel shows one row per server host', async () => {
            server.use(
                hostAccessHandlers.listKnownHosts.success([
                    createMockHost({ id: 1, fqdn: 'app.lan' }),
                    createMockHost({ id: 2, fqdn: 'db.lan' }),
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
                    createMockHost({ id: 1, fqdn: 'host.lan' }),
                ]),
            );
            renderHostsPage();

            await waitFor(() => {
                expect(screen.getByText('host.lan')).toBeInTheDocument();
            }, { timeout: TEST_TIMEOUTS.MEDIUM });

            expect(screen.queryByRole('button', { name: /save changes/i })).not.toBeInTheDocument();
        });
    });

    // ─── B5: Suggestions staging filter ─────────────────────────────────────────

    describe('suggestions staging', () => {
        it('staging a suggestion removes it from the list while leaving others', async () => {
            const user = setupUser();
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

            await waitFor(() => {
                expect(screen.getByText('foo.lan')).toBeInTheDocument();
                expect(screen.getByText('bar.lan')).toBeInTheDocument();
            }, { timeout: TEST_TIMEOUTS.MEDIUM });

            const fooRow = screen.getAllByRole('row').find((row) => within(row).queryByText('foo.lan'));
            await user.click(within(fooRow!).getByRole('button', { name: /add to known/i }));

            // A staged host moves into the known-hosts draft table, so scope the removal
            // assertion to the suggestions table (it has the "Allowed hits" column).
            await waitFor(() => {
                const suggestionsTable = screen.getByText('Allowed hits').closest('table')!;
                expect(within(suggestionsTable).queryByText('foo.lan')).not.toBeInTheDocument();
                expect(within(suggestionsTable).getByText('bar.lan')).toBeInTheDocument();
            }, { timeout: TEST_TIMEOUTS.SHORT });
        });

        it('staging a suggestion decrements the Suggestions badge and shows the StagedChangesBar', async () => {
            const user = setupUser();
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

            await waitFor(() => {
                const suggestionsToggle = screen.getByRole('button', { name: /observed in recent traffic/i });
                expect(within(suggestionsToggle).getByText('2')).toBeInTheDocument();
            }, { timeout: TEST_TIMEOUTS.MEDIUM });

            const fooRow = screen.getAllByRole('row').find((row) => within(row).queryByText('foo.lan'));
            await user.click(within(fooRow!).getByRole('button', { name: /add to known/i }));

            await waitFor(() => {
                const suggestionsToggle = screen.getByRole('button', { name: /observed in recent traffic/i });
                expect(within(suggestionsToggle).getByText('1')).toBeInTheDocument();
                expect(screen.getByRole('button', { name: /save changes/i })).toBeInTheDocument();
            }, { timeout: TEST_TIMEOUTS.MEDIUM });
        });
    });

    // ─── B6: Error state ─────────────────────────────────────────────────────────

    describe('error state', () => {
        it('shows error message when the suggestions query fails', async () => {
            server.use(hostAccessHandlers.listHostSuggestions.serverError());
            renderHostsPage();

            await waitFor(() => {
                expect(screen.getByText('Failed to load suggestions')).toBeInTheDocument();
                expect(screen.getByRole('button', { name: /try again/i })).toBeInTheDocument();
            }, { timeout: TEST_TIMEOUTS.MEDIUM });
        });
    });

    // ─── B7: Save flow smoke test ────────────────────────────────────────────────

    describe('save flow', () => {
        it('reconcile is called and the StagedChangesBar disappears after a successful save', async () => {
            const user = setupUser();
            const reconcileSpy = vi.fn();
            server.use(
                http.post(endpoints.adminHostsReconcile, () => {
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

            await waitFor(() => {
                expect(screen.getByRole('button', { name: /add host/i })).toBeInTheDocument();
            }, { timeout: TEST_TIMEOUTS.MEDIUM });

            await user.click(screen.getByRole('button', { name: /add host/i }));
            await waitFor(() => expect(screen.getByRole('dialog')).toBeInTheDocument());

            const dialog = screen.getByRole('dialog');
            await user.type(within(dialog).getByRole('textbox', { name: /fqdn/i }), 'new.lan');
            await user.click(within(dialog).getByRole('button', { name: /Add to draft/i }));

            await waitFor(() => {
                expect(screen.getByRole('button', { name: /save changes/i })).toBeInTheDocument();
            }, { timeout: TEST_TIMEOUTS.SHORT });

            await user.click(screen.getByRole('button', { name: /save changes/i }));

            await waitFor(() => expect(reconcileSpy).toHaveBeenCalledOnce(), { timeout: TEST_TIMEOUTS.LONG });
            await waitFor(() => {
                expect(screen.queryByRole('button', { name: /save changes/i })).not.toBeInTheDocument();
            }, { timeout: TEST_TIMEOUTS.MEDIUM });
        });
    });

    // ─── B8: useUnsavedChangesGuard hookup ──────────────────────────────────────

    describe('useUnsavedChangesGuard hookup', () => {
        it('is called with false initially and with true after staging a host', async () => {
            const user = setupUser();
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

            await waitFor(() => expect(screen.getByText('guard-test.lan')).toBeInTheDocument(), { timeout: TEST_TIMEOUTS.MEDIUM });
            await user.click(screen.getByRole('button', { name: /add to known/i }));

            await waitFor(() => {
                expect(mockGuard).toHaveBeenCalledWith(true);
            }, { timeout: TEST_TIMEOUTS.SHORT });
        });
    });
});
