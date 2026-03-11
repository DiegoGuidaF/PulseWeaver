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
import { useAdminUpdateUser } from './useAdminUpdateUser';

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

describe('useAdminUpdateUser', () => {
    it('shows success toast and invalidates user list and current user', async () => {
        server.use(
            handlers.auth.adminUpdateUserHandler({ role: 'admin' }),
            handlers.auth.listUsersHandler(),
            handlers.auth.meHandler()
        );

        const { queryClient, Wrapper } = createWrapper();
        queryClient.setQueryData(listUsersQueryKey(), [createMockUser()]);
        queryClient.setQueryData(getCurrentUserQueryKey(), createMockUser());

        const { result } = renderHook(() => useAdminUpdateUser(), { wrapper: Wrapper });

        act(() => {
            result.current.mutate({ path: { user_id: 2 }, body: { role: 'admin' } });
        });

        await waitFor(() => expect(result.current.isSuccess).toBe(true));

        expect(await screen.findByText('User updated')).toBeInTheDocument();
        expect(queryClient.getQueryState(listUsersQueryKey())?.isInvalidated).toBe(true);
        expect(queryClient.getQueryState(getCurrentUserQueryKey())?.isInvalidated).toBe(true);
    });

    it('shows error toast on failure', async () => {
        server.use(
            http.patch('/api/v1/admin/users/:userId', () =>
                responses.forbidden({ error: 'Forbidden role change' })
            )
        );

        const { Wrapper } = createWrapper();
        const { result } = renderHook(() => useAdminUpdateUser(), { wrapper: Wrapper });

        act(() => {
            result.current.mutate({ path: { user_id: 1 }, body: { role: 'user' } });
        });

        await waitFor(() => expect(result.current.isError).toBe(true));

        expect(await screen.findByText('Failed to update user')).toBeInTheDocument();
    });
});
