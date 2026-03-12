import { describe, expect, it } from 'vitest';
import { renderHook, act, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { http } from 'msw';
import React from 'react';
import { server } from '@/test/setup';
import { authHandlers, endpoints, responses } from '@/test/mocks/handlers';
import { createMockUser } from '@/test/mocks/data';
import { getCurrentUserQueryKey, listUsersQueryKey } from '@/lib/api/@tanstack/react-query.gen';
import { useAdminUpdateUser } from './useAdminUpdateUser';

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

describe('useAdminUpdateUser', () => {
    it('invalidates user list and current user on success', async () => {
        server.use(
            authHandlers.adminUpdateUser.success({ role: 'admin' }),
            // authHandlers.listUsers.success() and authHandlers.me.success() are in defaultHandlers
        );

        const { queryClient, Wrapper } = createWrapper();
        queryClient.setQueryData(listUsersQueryKey(), [createMockUser()]);
        queryClient.setQueryData(getCurrentUserQueryKey(), createMockUser());

        const { result } = renderHook(() => useAdminUpdateUser(), { wrapper: Wrapper });

        act(() => {
            result.current.mutate({ path: { user_id: 2 }, body: { role: 'admin' } });
        });

        await waitFor(() => expect(result.current.isSuccess).toBe(true));

        expect(queryClient.getQueryState(listUsersQueryKey())?.isInvalidated).toBe(true);
        expect(queryClient.getQueryState(getCurrentUserQueryKey())?.isInvalidated).toBe(true);
    });

    it('enters error state on failure', async () => {
        server.use(
            http.patch(endpoints.adminUserById, () =>
                responses.forbidden({ error: 'Forbidden role change' })
            )
        );

        const { Wrapper } = createWrapper();
        const { result } = renderHook(() => useAdminUpdateUser(), { wrapper: Wrapper });

        act(() => {
            result.current.mutate({ path: { user_id: 1 }, body: { role: 'user' } });
        });

        await waitFor(() => expect(result.current.isError).toBe(true));
    });
});
