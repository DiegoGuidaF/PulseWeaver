import { describe, expect, it } from 'vitest';
import { screen, waitFor } from '@testing-library/react';
import { NetworkPoliciesPage } from '@/pages/access/network-policies/NetworkPoliciesPage';
import { TEST_TIMEOUTS } from '@/test/constants';
import { networkPolicyHandlers } from '@/test/mocks/handlers';
import { server } from '@/test/setup';
import { renderWithProviders } from '@/test/utils';
import { createMockNetworkPolicyListItem } from '@/test/mocks/data';

describe('NetworkPoliciesPage', () => {
    describe('without group filter', () => {
        it('renders heading, new policy button, and all policies', async () => {
            server.use(
                networkPolicyHandlers.list.success([
                    createMockNetworkPolicyListItem({ id: 1, name: 'Allow VPN' }),
                    createMockNetworkPolicyListItem({ id: 2, name: 'Block External' }),
                ]),
            );

            renderWithProviders(<NetworkPoliciesPage />);

            await waitFor(
                () => {
                    expect(screen.getByRole('heading', { name: 'Network Policies', level: 1 })).toBeInTheDocument();
                    expect(screen.getByRole('button', { name: /new policy/i })).toBeInTheDocument();
                    expect(screen.getByText('Allow VPN')).toBeInTheDocument();
                    expect(screen.getByText('Block External')).toBeInTheDocument();
                },
                { timeout: TEST_TIMEOUTS.SHORT },
            );
        });
    });

    describe('with ?group_id URL param', () => {
        it('shows only policies belonging to that group', async () => {
            server.use(
                networkPolicyHandlers.list.success([
                    createMockNetworkPolicyListItem({
                        id: 1,
                        name: 'Policy In Group',
                        groups: [{ id: 5, name: 'Engineering' }],
                    }),
                    createMockNetworkPolicyListItem({
                        id: 2,
                        name: 'Policy Not In Group',
                        groups: [],
                    }),
                ]),
            );

            renderWithProviders(
                <NetworkPoliciesPage />,
                { initialEntries: ['/access/network-policies?group_id=5'] },
            );

            await waitFor(
                () => expect(screen.getByText('Policy In Group')).toBeInTheDocument(),
                { timeout: TEST_TIMEOUTS.SHORT },
            );

            expect(screen.queryByText('Policy Not In Group')).not.toBeInTheDocument();
        });

        it('shows all policies when ?group_id does not match any', async () => {
            server.use(
                networkPolicyHandlers.list.success([
                    createMockNetworkPolicyListItem({ id: 1, name: 'Alpha', groups: [] }),
                ]),
            );

            renderWithProviders(
                <NetworkPoliciesPage />,
                { initialEntries: ['/access/network-policies?group_id=99'] },
            );

            await waitFor(
                () => expect(screen.getByRole('heading', { name: 'Network Policies', level: 1 })).toBeInTheDocument(),
                { timeout: TEST_TIMEOUTS.SHORT },
            );

            expect(screen.queryByText('Alpha')).not.toBeInTheDocument();
        });
    });
});
