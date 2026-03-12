import { describe, expect, it } from 'vitest';
import { renderHook, act, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import React from 'react';
import { http, HttpResponse } from 'msw';
import { server } from '@/test/setup';
import { authHandlers, endpoints, responses } from '@/test/mocks/handlers';
import { createMockUser } from '@/test/mocks/data';
import { getCurrentUserQueryKey } from '@/lib/api/@tanstack/react-query.gen';
import { useUpdateMe } from './useUpdateMe';

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

describe('useUpdateMe', () => {
    it('invalidates current user on success', async () => {
        server.use(
            authHandlers.updateMe.success({ display_name: 'Updated Name' }),
            // authHandlers.me.success() is in defaultHandlers
        );

        const { queryClient, Wrapper } = createWrapper();
        queryClient.setQueryData(getCurrentUserQueryKey(), createMockUser());

        const { result } = renderHook(() => useUpdateMe(), { wrapper: Wrapper });

        act(() => {
            result.current.mutate({ body: { display_name: 'Updated Name' } });
        });

        await waitFor(() => expect(result.current.isSuccess).toBe(true));

        expect(queryClient.getQueryState(getCurrentUserQueryKey())?.isInvalidated).toBe(true);
    });

    it('enters error state on 409 conflict', async () => {
        server.use(
            http.patch(endpoints.updateMe, () =>
                HttpResponse.json({ error: 'Conflict' }, { status: 409 })
            )
        );

        const { Wrapper } = createWrapper();
        const { result } = renderHook(() => useUpdateMe(), { wrapper: Wrapper });

        act(() => {
            result.current.mutate({ body: { username: 'taken_name' } });
        });

        await waitFor(() => expect(result.current.isError).toBe(true));
    });

    it('enters error state on server error', async () => {
        server.use(
            http.patch(endpoints.updateMe, () => responses.serverError())
        );

        const { Wrapper } = createWrapper();
        const { result } = renderHook(() => useUpdateMe(), { wrapper: Wrapper });

        act(() => {
            result.current.mutate({ body: { display_name: 'Fail' } });
        });

        await waitFor(() => expect(result.current.isError).toBe(true));
    });
});
