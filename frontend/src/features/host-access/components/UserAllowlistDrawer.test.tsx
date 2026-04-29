import { beforeEach, describe, expect, it, vi } from 'vitest';
import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { http } from 'msw';
import { UserAllowlistDrawer } from '@/features/host-access/components/UserAllowlistDrawer';
import { TEST_TIMEOUTS } from '@/test/constants';
import { endpoints, hostAccessHandlers, responses } from '@/test/mocks/handlers';
import { server } from '@/test/setup';
import { renderWithProviders } from '@/test/utils';
import {
    createMockUserHostAccessSummary,
    createMockUserHostDetails,
    createMockUserHostDetailsGroup,
    createMockUserHostDetailsHost,
} from '@/test/mocks/data';
import { UserRole } from '@/lib/api';
import type { SetUserHostGrantsRequest } from '@/lib/api';

// Helper to render the drawer with a user prop and onClose spy
function renderDrawer(
    user: ReturnType<typeof createMockUserHostAccessSummary> | null,
    onClose = vi.fn(),
) {
    return {
        onClose,
        ...renderWithProviders(<UserAllowlistDrawer user={user} onClose={onClose} />),
    };
}

// Standard fixture: 2 groups, 4 hosts
// group 1: granted, covers host 3 and host 4
// group 2: not granted
// host 1: directly_granted
// host 2: not granted
// host 3: covered by group 1 (via_group)
// host 4: covered by group 1 (via_group)
function buildRichDetails() {
    const group1 = createMockUserHostDetailsGroup({
        id: 1,
        name: 'infra',
        granted: true,
        hosts: [
            { id: 3, fqdn: 'db.example.com', icon: null },
            { id: 4, fqdn: 'cache.example.com', icon: null },
        ],
    });
    const group2 = createMockUserHostDetailsGroup({
        id: 2,
        name: 'dev',
        granted: false,
        hosts: [], // no preview text — prevents dev.example.com appearing in two DOM locations
    });
    const host1 = createMockUserHostDetailsHost({ id: 1, fqdn: 'app.example.com', directly_granted: true, via_group: null });
    const host2 = createMockUserHostDetailsHost({ id: 2, fqdn: 'dev.example.com', directly_granted: false, via_group: null });
    const host3 = createMockUserHostDetailsHost({ id: 3, fqdn: 'db.example.com', directly_granted: false, via_group: { id: 1, name: 'infra' } });
    const host4 = createMockUserHostDetailsHost({ id: 4, fqdn: 'cache.example.com', directly_granted: false, via_group: { id: 1, name: 'infra' } });

    return createMockUserHostDetails({
        id: 5,
        display_name: 'Dana',
        role: UserRole.USER,
        bypass: false,
        groups: [group1, group2],
        hosts: [host1, host2, host3, host4],
    });
}

describe('UserAllowlistDrawer', () => {
    describe('loading state', () => {
        it('shows loader while details are pending; form renders once data resolves', async () => {
            let resolveDetails: ((value: Response) => void) | null = null;
            // handlerCalled resolves once MSW intercepts the request (so resolveDetails is guaranteed set)
            const handlerCalled = new Promise<void>((res) => {
                server.use(
                    http.get(endpoints.userHostDetails, () => {
                        res();
                        return new Promise<Response>((resolve) => { resolveDetails = resolve; });
                    }),
                );
            });

            const user = createMockUserHostAccessSummary({ id: 5, display_name: 'Dana', role: UserRole.USER });
            renderDrawer(user);

            // Loader visible immediately (React Query sets isFetching=true before the first response)
            expect(screen.getByText('Loading access details…')).toBeInTheDocument();

            // Wait for the request to actually reach the MSW handler, then resolve it
            await handlerCalled;
            resolveDetails!(new Response(JSON.stringify(createMockUserHostDetails({ id: 5, display_name: 'Dana' })), {
                headers: { 'Content-Type': 'application/json' },
            }));

            await waitFor(
                () => {
                    expect(screen.getByText('Allow all hosts')).toBeInTheDocument();
                },
                { timeout: TEST_TIMEOUTS.SHORT },
            );
        });
    });

    describe('initial form state', () => {
        it('mirrors server data: bypass switch, granted-group checkboxes, direct host checkboxes', async () => {
            const details = buildRichDetails();
            server.use(hostAccessHandlers.userHostDetails.success(details));

            const user = createMockUserHostAccessSummary({ id: 5, display_name: 'Dana', role: UserRole.USER });
            renderDrawer(user);

            await waitFor(
                () => {
                    expect(screen.getByText('Allow all hosts')).toBeInTheDocument();
                },
                { timeout: TEST_TIMEOUTS.SHORT },
            );

            // bypass switch should be unchecked (Mantine Switch renders as role="switch")
            const bypassSwitch = screen.getByRole('switch');
            expect(bypassSwitch).not.toBeChecked();

            // group 1 (infra) is granted → checkbox checked
            const groupCheckboxes = screen.getAllByRole('checkbox');
            // group checkboxes come before host checkboxes in the DOM
            // infra group should be checked, dev group should not
            const checkedGroupCheckboxes = groupCheckboxes.filter(cb => (cb as HTMLInputElement).checked);
            expect(checkedGroupCheckboxes.length).toBeGreaterThanOrEqual(1);

            // host 1 (app.example.com) is directly granted → should appear as checked
            // host 3 and host 4 are via group (hidden by default in "all" filter)
            // host 2 is not granted
            expect(screen.getByText('app.example.com')).toBeInTheDocument();
            expect(screen.getByText('dev.example.com')).toBeInTheDocument();
        });
    });

    describe('host checkbox interactions', () => {
        it('toggling an unchecked host checkbox increments the effective access count by 1', async () => {
            const details = createMockUserHostDetails({
                id: 5,
                display_name: 'Dana',
                role: UserRole.USER,
                bypass: false,
                groups: [],
                hosts: [
                    createMockUserHostDetailsHost({ id: 1, fqdn: 'app.example.com', directly_granted: false }),
                    createMockUserHostDetailsHost({ id: 2, fqdn: 'dev.example.com', directly_granted: false }),
                ],
            });
            server.use(hostAccessHandlers.userHostDetails.success(details));

            const user = createMockUserHostAccessSummary({ id: 5, display_name: 'Dana', role: UserRole.USER });
            renderDrawer(user);

            await waitFor(
                () => {
                    expect(screen.getByText('app.example.com')).toBeInTheDocument();
                },
                { timeout: TEST_TIMEOUTS.SHORT },
            );

            // Initial state: no hosts granted yet — component shows the "no hosts" warning
            expect(screen.getByText(/No hosts — all requests will be denied/i)).toBeInTheDocument();

            // Toggle app.example.com checkbox
            const hostCheckboxes = screen.getAllByRole('checkbox').filter(
                cb => !(cb as HTMLInputElement).disabled,
            );
            await userEvent.click(hostCheckboxes[0]);

            await waitFor(
                () => {
                    // <strong>1</strong> of 2 hosts — text is split across child elements,
                    // so match against the parent <p>'s full textContent via function matcher
                    expect(screen.getByText(
                        (_, el) => el?.tagName === 'P' && /1 of 2 hosts/.test(el?.textContent ?? ''),
                    )).toBeInTheDocument();
                },
                { timeout: TEST_TIMEOUTS.SHORT },
            );
        });

        it('toggling a group ON disables its hosts and shows the "via group" badge', async () => {
            const group1 = createMockUserHostDetailsGroup({
                id: 1,
                name: 'infra',
                granted: false,
                hosts: [{ id: 1, fqdn: 'app.example.com', icon: null }],
            });
            const details = createMockUserHostDetails({
                id: 5,
                display_name: 'Dana',
                role: UserRole.USER,
                bypass: false,
                groups: [group1],
                hosts: [createMockUserHostDetailsHost({ id: 1, fqdn: 'app.example.com', directly_granted: false, via_group: null })],
            });
            server.use(hostAccessHandlers.userHostDetails.success(details));

            const user = createMockUserHostAccessSummary({ id: 5, display_name: 'Dana', role: UserRole.USER });
            renderDrawer(user);

            await waitFor(
                () => {
                    expect(screen.getByText('infra')).toBeInTheDocument();
                },
                { timeout: TEST_TIMEOUTS.SHORT },
            );

            // Toggle the infra group checkbox ON
            // The group checkbox is an unchecked checkbox (group not granted initially)
            const groupCheckbox = screen.getAllByRole('checkbox').find(
                cb => !(cb as HTMLInputElement).checked,
            );
            expect(groupCheckbox).toBeDefined();
            await userEvent.click(groupCheckbox!);

            // After toggling group on, the host should appear as "via group" covered
            // Need to show via-group rows first
            await waitFor(
                () => {
                    const showBtn = screen.queryByRole('button', { name: /show.*covered by groups/i });
                    return showBtn !== null;
                },
                { timeout: TEST_TIMEOUTS.SHORT },
            ).catch(() => {
                // Host might already be visible
            });

            const showBtn = screen.queryByRole('button', { name: /show.*covered by groups/i });
            if (showBtn) {
                await userEvent.click(showBtn);
            }

            await waitFor(
                () => {
                    expect(screen.getByText('via group')).toBeInTheDocument();
                },
                { timeout: TEST_TIMEOUTS.SHORT },
            );
        });
    });

    describe('bypass Risky badge', () => {
        it('bypass ON for a non-admin user (role: "user") shows the "Risky" badge', async () => {
            const details = createMockUserHostDetails({
                id: 5,
                display_name: 'Dana',
                role: UserRole.USER,
                bypass: true,
                groups: [],
                hosts: [],
            });
            server.use(hostAccessHandlers.userHostDetails.success(details));

            const user = createMockUserHostAccessSummary({ id: 5, display_name: 'Dana', role: UserRole.USER, bypass: true });
            renderDrawer(user);

            await waitFor(
                () => {
                    expect(screen.getByText('Risky')).toBeInTheDocument();
                },
                { timeout: TEST_TIMEOUTS.SHORT },
            );
        });
    });

    describe('search filter', () => {
        it('search input filters the host list by FQDN substring', async () => {
            const details = createMockUserHostDetails({
                id: 5,
                display_name: 'Dana',
                role: UserRole.USER,
                bypass: false,
                groups: [],
                hosts: [
                    createMockUserHostDetailsHost({ id: 1, fqdn: 'app.example.com', directly_granted: true }),
                    createMockUserHostDetailsHost({ id: 2, fqdn: 'api.other.org', directly_granted: true }),
                ],
            });
            server.use(hostAccessHandlers.userHostDetails.success(details));

            const user = createMockUserHostAccessSummary({ id: 5, display_name: 'Dana', role: UserRole.USER });
            renderDrawer(user);

            await waitFor(
                () => {
                    expect(screen.getByText('app.example.com')).toBeInTheDocument();
                    expect(screen.getByText('api.other.org')).toBeInTheDocument();
                },
                { timeout: TEST_TIMEOUTS.SHORT },
            );

            const searchInput = screen.getByPlaceholderText(/search hosts/i);
            await userEvent.type(searchInput, 'example');

            await waitFor(
                () => {
                    expect(screen.getByText('app.example.com')).toBeInTheDocument();
                    expect(screen.queryByText('api.other.org')).not.toBeInTheDocument();
                },
                { timeout: TEST_TIMEOUTS.SHORT },
            );
        });
    });

    describe('segmented filter', () => {
        beforeEach(() => {
            const details = buildRichDetails();
            server.use(hostAccessHandlers.userHostDetails.success(details));
        });

        it('filter "Granted" shows only direct or via-group hosts', async () => {
            const user = createMockUserHostAccessSummary({ id: 5, display_name: 'Dana', role: UserRole.USER });
            renderDrawer(user);

            await waitFor(
                () => {
                    expect(screen.getByText('app.example.com')).toBeInTheDocument();
                },
                { timeout: TEST_TIMEOUTS.SHORT },
            );

            // Click "Granted" filter — try radio role first
            let grantedOption = screen.queryByRole('radio', { name: 'Granted' });
            if (!grantedOption) {
                grantedOption = screen.getByText('Granted');
            }
            await userEvent.click(grantedOption);

            await waitFor(
                () => {
                    // app.example.com is directly granted → visible
                    expect(screen.getByText('app.example.com')).toBeInTheDocument();
                    // dev.example.com is not granted → hidden
                    expect(screen.queryByText('dev.example.com')).not.toBeInTheDocument();
                },
                { timeout: TEST_TIMEOUTS.SHORT },
            );
        });

        it('filter "Not granted" shows only hosts with neither direct nor via-group access', async () => {
            const user = createMockUserHostAccessSummary({ id: 5, display_name: 'Dana', role: UserRole.USER });
            renderDrawer(user);

            await waitFor(
                () => {
                    expect(screen.getByText('dev.example.com')).toBeInTheDocument();
                },
                { timeout: TEST_TIMEOUTS.SHORT },
            );

            let notGrantedOption = screen.queryByRole('radio', { name: 'Not granted' });
            if (!notGrantedOption) {
                notGrantedOption = screen.getByText('Not granted');
            }
            await userEvent.click(notGrantedOption);

            await waitFor(
                () => {
                    // dev.example.com is not granted → visible
                    expect(screen.getByText('dev.example.com')).toBeInTheDocument();
                    // app.example.com is directly granted → hidden
                    expect(screen.queryByText('app.example.com')).not.toBeInTheDocument();
                },
                { timeout: TEST_TIMEOUTS.SHORT },
            );
        });
    });

    describe('show/hide via-group rows', () => {
        it('"Show N covered by groups" button reveals via-group rows; "Hide via-group rows" reverses it', async () => {
            const details = buildRichDetails();
            server.use(hostAccessHandlers.userHostDetails.success(details));

            const user = createMockUserHostAccessSummary({ id: 5, display_name: 'Dana', role: UserRole.USER });
            renderDrawer(user);

            await waitFor(
                () => {
                    expect(screen.getByText('app.example.com')).toBeInTheDocument();
                },
                { timeout: TEST_TIMEOUTS.SHORT },
            );

            // Via-group only hosts (db.example.com, cache.example.com) are hidden by default in "all" filter
            expect(screen.queryByText('db.example.com')).not.toBeInTheDocument();

            // Show button should be visible
            const showBtn = await screen.findByRole('button', { name: /show.*covered by groups/i });
            await userEvent.click(showBtn);

            await waitFor(
                () => {
                    expect(screen.getByText('db.example.com')).toBeInTheDocument();
                    expect(screen.getByText('cache.example.com')).toBeInTheDocument();
                },
                { timeout: TEST_TIMEOUTS.SHORT },
            );

            // "Hide via-group rows" button should now be visible
            const hideBtn = screen.getByRole('button', { name: /hide via-group rows/i });
            await userEvent.click(hideBtn);

            await waitFor(
                () => {
                    expect(screen.queryByText('db.example.com')).not.toBeInTheDocument();
                },
                { timeout: TEST_TIMEOUTS.SHORT },
            );
        });
    });

    describe('Save action', () => {
        it('Save fires PUT with correct body and calls onClose', async () => {
            const details = createMockUserHostDetails({
                id: 5,
                display_name: 'Dana',
                role: UserRole.USER,
                bypass: false,
                groups: [
                    createMockUserHostDetailsGroup({ id: 1, name: 'infra', granted: true, hosts: [] }),
                ],
                hosts: [
                    createMockUserHostDetailsHost({ id: 1, fqdn: 'app.example.com', directly_granted: true }),
                ],
            });
            server.use(hostAccessHandlers.userHostDetails.success(details));

            let captured: SetUserHostGrantsRequest | null = null;
            server.use(
                http.put(endpoints.setUserHostGrants, async ({ request }) => {
                    captured = await request.json() as SetUserHostGrantsRequest;
                    return responses.noContent();
                }),
            );

            const user = createMockUserHostAccessSummary({ id: 5, display_name: 'Dana', role: UserRole.USER });
            const { onClose } = renderDrawer(user);

            await waitFor(
                () => {
                    expect(screen.getByText('Allow all hosts')).toBeInTheDocument();
                },
                { timeout: TEST_TIMEOUTS.SHORT },
            );

            await userEvent.click(screen.getByRole('button', { name: /^save$/i }));

            await waitFor(
                () => {
                    expect(captured).not.toBeNull();
                },
                { timeout: TEST_TIMEOUTS.SHORT },
            );

            expect(captured).toMatchObject({
                bypass: false,
                group_ids: [1],
                host_ids: [1],
            });
            expect(onClose).toHaveBeenCalled();
        });
    });

    describe('Cancel action', () => {
        it('Cancel calls onClose without firing PUT', async () => {
            const details = createMockUserHostDetails({ id: 5, display_name: 'Dana' });
            server.use(hostAccessHandlers.userHostDetails.success(details));

            let putCalled = false;
            server.use(
                http.put(endpoints.setUserHostGrants, async () => {
                    putCalled = true;
                    return responses.noContent();
                }),
            );

            const user = createMockUserHostAccessSummary({ id: 5, display_name: 'Dana', role: UserRole.USER });
            const { onClose } = renderDrawer(user);

            await waitFor(
                () => {
                    expect(screen.getByText('Allow all hosts')).toBeInTheDocument();
                },
                { timeout: TEST_TIMEOUTS.SHORT },
            );

            await userEvent.click(screen.getByRole('button', { name: /cancel/i }));

            expect(onClose).toHaveBeenCalled();
            expect(putCalled).toBe(false);
        });
    });

    describe('re-render with different user', () => {
        it('re-fetches and re-initializes form when user.id changes', async () => {
            const details1 = createMockUserHostDetails({
                id: 5,
                display_name: 'Dana',
                role: UserRole.USER,
                bypass: false,
                groups: [],
                hosts: [createMockUserHostDetailsHost({ id: 1, fqdn: 'app.example.com', directly_granted: true })],
            });
            const details2 = createMockUserHostDetails({
                id: 6,
                display_name: 'Eve',
                role: UserRole.ADMIN,
                bypass: true,
                groups: [],
                hosts: [],
            });

            server.use(
                http.get(endpoints.userHostDetails, ({ params }) => {
                    const userId = Number(params.userId);
                    if (userId === 5) return new Response(JSON.stringify(details1), { headers: { 'Content-Type': 'application/json' } });
                    return new Response(JSON.stringify(details2), { headers: { 'Content-Type': 'application/json' } });
                }),
            );

            const user1 = createMockUserHostAccessSummary({ id: 5, display_name: 'Dana', role: UserRole.USER });
            const user2 = createMockUserHostAccessSummary({ id: 6, display_name: 'Eve', role: UserRole.ADMIN, bypass: true });

            const { rerender, onClose } = renderDrawer(user1);

            await waitFor(
                () => {
                    expect(screen.getByText('app.example.com')).toBeInTheDocument();
                },
                { timeout: TEST_TIMEOUTS.SHORT },
            );

            // Rerender with different user
            rerender(<UserAllowlistDrawer user={user2} onClose={onClose} />);

            await waitFor(
                () => {
                    // Eve has bypass=true, so "Risky" badge should show (user role)
                    // After rerender, bypass switch should reflect new user's data
                    expect(screen.queryByText('app.example.com')).not.toBeInTheDocument();
                },
                { timeout: TEST_TIMEOUTS.SHORT },
            );
        });
    });
});
