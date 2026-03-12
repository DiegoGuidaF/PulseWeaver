import { describe, expect, it } from 'vitest';
import { renderHook, act, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { http } from 'msw';
import React from 'react';
import { server } from '@/test/setup';
import { endpoints, responses } from '@/test/mocks/handlers';
import { createMockUser } from '@/test/mocks/data';
import { getCurrentUserQueryKey } from '@/lib/api/@tanstack/react-query.gen';
import { useChangePassword } from './useChangePassword';

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

describe('useChangePassword', () => {
    it('invalidates current user on success', async () => {
        // authHandlers.changePassword.success() and authHandlers.me.success() are in defaultHandlers

        const { queryClient, Wrapper } = createWrapper();
        queryClient.setQueryData(getCurrentUserQueryKey(), createMockUser());

        const { result } = renderHook(() => useChangePassword(), { wrapper: Wrapper });

        act(() => {
            result.current.mutate({ body: { current_password: 'OldPass!', password: 'NewPass123!' } });
        });

        await waitFor(() => expect(result.current.isSuccess).toBe(true));

        expect(queryClient.getQueryState(getCurrentUserQueryKey())?.isInvalidated).toBe(true);
    });

    it('enters error state on wrong current password', async () => {
        server.use(
            http.post(endpoints.changePassword, () =>
                responses.badRequest({ error: 'Invalid password change request' })
            )
        );

        const { Wrapper } = createWrapper();
        const { result } = renderHook(() => useChangePassword(), { wrapper: Wrapper });

        act(() => {
            result.current.mutate({ body: { current_password: 'WrongPass!', password: 'NewPass123!' } });
        });

        await waitFor(() => expect(result.current.isError).toBe(true));
    });
});
