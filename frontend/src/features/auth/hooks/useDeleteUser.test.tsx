import { describe, expect, it } from 'vitest';
import { renderHook, act, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { http } from 'msw';
import React from 'react';
import { server } from '@/test/setup';
import { endpoints, responses } from '@/test/mocks/handlers';
import { createMockUser } from '@/test/mocks/data';
import { getCurrentUserQueryKey, listUsersQueryKey } from '@/lib/api/@tanstack/react-query.gen';
import { useDeleteUser } from './useDeleteUser';

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

describe('useDeleteUser', () => {
    it('invalidates user list and current user on success', async () => {
        // authHandlers.deleteUser.success(), listUsers.success(), me.success() are in defaultHandlers

        const { queryClient, Wrapper } = createWrapper();
        queryClient.setQueryData(listUsersQueryKey(), [createMockUser()]);
        queryClient.setQueryData(getCurrentUserQueryKey(), createMockUser());

        const { result } = renderHook(() => useDeleteUser(), { wrapper: Wrapper });

        act(() => {
            result.current.mutate({ path: { user_id: 2 } });
        });

        await waitFor(() => expect(result.current.isSuccess).toBe(true));

        expect(queryClient.getQueryState(listUsersQueryKey())?.isInvalidated).toBe(true);
        expect(queryClient.getQueryState(getCurrentUserQueryKey())?.isInvalidated).toBe(true);
    });

    it('enters error state when delete is forbidden', async () => {
        server.use(
            http.delete(endpoints.adminUserById, () =>
                responses.forbidden({ error: 'Forbidden user delete' })
            )
        );

        const { Wrapper } = createWrapper();
        const { result } = renderHook(() => useDeleteUser(), { wrapper: Wrapper });

        act(() => {
            result.current.mutate({ path: { user_id: 1 } });
        });

        await waitFor(() => expect(result.current.isError).toBe(true));
    });

    it('enters error state when user is not found', async () => {
        server.use(
            http.delete(endpoints.adminUserById, () =>
                responses.notFound({ error: 'User not found' })
            )
        );

        const { Wrapper } = createWrapper();
        const { result } = renderHook(() => useDeleteUser(), { wrapper: Wrapper });

        act(() => {
            result.current.mutate({ path: { user_id: 99 } });
        });

        await waitFor(() => expect(result.current.isError).toBe(true));
    });
});
