import { describe, expect, it } from 'vitest';
import { renderHook, act, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import React from 'react';
import { fleetListKey, fleetOwnerKey } from '@/features/devices/fleetCache';
import { useSetUserAccess } from './useSetUserAccess';

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

describe('useSetUserAccess', () => {
    it('invalidates the fleet, which renders bypass and group badges from its own copy', async () => {
        // setUserHostGrants.success() is in defaultHandlers.

        const { queryClient, Wrapper } = createWrapper();
        queryClient.setQueryData(fleetListKey(), []);
        queryClient.setQueryData(fleetOwnerKey(7), []);

        const { result } = renderHook(() => useSetUserAccess(), { wrapper: Wrapper });

        act(() => {
            result.current.mutate({
                path: { user_id: 7 },
                body: { bypass_host_check: true, group_ids: [] },
            });
        });

        await waitFor(() => expect(result.current.isSuccess).toBe(true));

        // The "All hosts" badge and the group badges are owner-identity fields, so
        // both the list and every cached owner entry go stale.
        expect(queryClient.getQueryState(fleetListKey())?.isInvalidated).toBe(true);
        expect(queryClient.getQueryState(fleetOwnerKey(7))?.isInvalidated).toBe(true);
    });
});
