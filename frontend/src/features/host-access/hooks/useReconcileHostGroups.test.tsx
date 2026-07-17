import { describe, expect, it } from 'vitest';
import { renderHook, act, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import React from 'react';
import { createMockGroupDetailWithUsers } from '@/test/mocks/data';
import { getHostGroupQueryKey, listNetworkPoliciesQueryKey } from '@/lib/api/@tanstack/react-query.gen';
import { useReconcileHostGroups } from './useReconcileHostGroups';

function createWrapper() {
    const queryClient = new QueryClient({
        defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
    });
    function Wrapper({ children }: { children: React.ReactNode }) {
        return (
            <QueryClientProvider client={queryClient}>
                {children}
            </QueryClientProvider>
        );
    }
    return { queryClient, Wrapper };
}

describe('useReconcileHostGroups', () => {
    it('invalidates every cached group detail and the network-policies list on success', async () => {
        // reconcileHostGroups.success() is in defaultHandlers.

        const { queryClient, Wrapper } = createWrapper();
        const detailKey = getHostGroupQueryKey({ path: { group_id: 1 } });
        queryClient.setQueryData(detailKey, createMockGroupDetailWithUsers({ id: 1 }));
        queryClient.setQueryData(listNetworkPoliciesQueryKey(), []);

        const { result } = renderHook(() => useReconcileHostGroups(), { wrapper: Wrapper });

        act(() => {
            result.current.mutate({ body: { groups: [] } });
        });

        await waitFor(() => expect(result.current.isSuccess).toBe(true));

        // Selected-group detail — the "invalidate nothing here" gap this bundle fixes.
        expect(queryClient.getQueryState(detailKey)?.isInvalidated).toBe(true);
        // Network policies' host_count / member lists go stale too (task 3b).
        expect(queryClient.getQueryState(listNetworkPoliciesQueryKey())?.isInvalidated).toBe(true);
    });
});
