import { describe, expect, it } from 'vitest';
import { renderHook, act, waitFor, screen } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { Toaster } from 'sonner';
import { http, HttpResponse } from 'msw';
import React from 'react';
import { server } from '@/test/setup';
import { handlers, responses } from '@/test/mocks/handlers';
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
                <Toaster />
                {children}
            </QueryClientProvider>
        );
    }
    return { queryClient, Wrapper };
}

describe('useDeleteUser', () => {
    it('shows success toast and invalidates user list and current user', async () => {
        server.use(
            handlers.auth.deleteUserHandler,
            handlers.auth.listUsersHandler(),
            handlers.auth.meHandler()
        );

        const { queryClient, Wrapper } = createWrapper();
        queryClient.setQueryData(listUsersQueryKey(), [createMockUser()]);
        queryClient.setQueryData(getCurrentUserQueryKey(), createMockUser());

        const { result } = renderHook(() => useDeleteUser(), { wrapper: Wrapper });

        act(() => {
            result.current.mutate({ path: { user_id: 2 } });
        });

        await waitFor(() => expect(result.current.isSuccess).toBe(true));

        expect(await screen.findByText('User deleted')).toBeInTheDocument();
        expect(queryClient.getQueryState(listUsersQueryKey())?.isInvalidated).toBe(true);
        expect(queryClient.getQueryState(getCurrentUserQueryKey())?.isInvalidated).toBe(true);
    });

    it('shows error toast when delete is forbidden', async () => {
        server.use(
            http.delete('/api/v1/admin/users/:userId', () =>
                responses.forbidden({ error: 'Forbidden user delete' })
            )
        );

        const { Wrapper } = createWrapper();
        const { result } = renderHook(() => useDeleteUser(), { wrapper: Wrapper });

        act(() => {
            result.current.mutate({ path: { user_id: 1 } });
        });

        await waitFor(() => expect(result.current.isError).toBe(true));

        expect(await screen.findByText('Failed to delete user')).toBeInTheDocument();
    });

    it('shows error toast when user is not found', async () => {
        server.use(
            http.delete('/api/v1/admin/users/:userId', () =>
                responses.notFound({ error: 'User not found' })
            )
        );

        const { Wrapper } = createWrapper();
        const { result } = renderHook(() => useDeleteUser(), { wrapper: Wrapper });

        act(() => {
            result.current.mutate({ path: { user_id: 99 } });
        });

        await waitFor(() => expect(result.current.isError).toBe(true));

        expect(await screen.findByText('Failed to delete user')).toBeInTheDocument();
    });
});
