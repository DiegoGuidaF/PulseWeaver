import { describe, expect, it } from 'vitest';
import { renderHook, act, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import React from 'react';
import { fleetListKey, fleetOwnerKey } from '@/features/devices/fleetCache';
import { useReconcileHosts } from './useReconcileHosts';

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

describe('useReconcileHosts', () => {
    it('invalidates the fleet, whose owner rows carry their own copy of host_groups[]', async () => {
        // reconcileKnownHosts.success() is in defaultHandlers.

        const { queryClient, Wrapper } = createWrapper();
        queryClient.setQueryData(fleetListKey(), []);
        queryClient.setQueryData(fleetOwnerKey(1), []);

        const { result } = renderHook(() => useReconcileHosts(), { wrapper: Wrapper });

        act(() => {
            result.current.mutate({ body: { hosts: [] } });
        });

        await waitFor(() => expect(result.current.isSuccess).toBe(true));

        // Moving a host between groups shifts the badges the fleet renders from its
        // own cached copy; without this they stay stale for FLEET_STALE_TIME.
        expect(queryClient.getQueryState(fleetListKey())?.isInvalidated).toBe(true);
        expect(queryClient.getQueryState(fleetOwnerKey(1))?.isInvalidated).toBe(true);
    });
});
