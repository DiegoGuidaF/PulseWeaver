import { describe, expect, it } from 'vitest';
import { renderHook, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import React from 'react';
import { server } from '@/test/setup';
import { handlers, responses } from '@/test/mocks/handlers';
import { createMockUser } from '@/test/mocks/data';
import { useListUsers } from './useListUsers';

function createWrapper() {
    const queryClient = new QueryClient({
        defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
    });
    function Wrapper({ children }: { children: React.ReactNode }) {
        return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>;
    }
    return { queryClient, Wrapper };
}

describe('useListUsers', () => {
    it('returns user list on success', async () => {
        server.use(
            handlers.auth.listUsersHandler([
                createMockUser({ id: 1, username: 'alice' }),
                createMockUser({ id: 2, username: 'bob' }),
            ])
        );

        const { Wrapper } = createWrapper();
        const { result } = renderHook(() => useListUsers(), { wrapper: Wrapper });

        await waitFor(() => expect(result.current.isSuccess).toBe(true));

        expect(result.current.data).toHaveLength(2);
        expect(result.current.data![0].username).toBe('alice');
        expect(result.current.data![1].username).toBe('bob');
    });

    it('exposes error state when the API fails', async () => {
        server.use(
            handlers.auth.listUsersHandler(undefined, () => responses.serverError())
        );

        const { Wrapper } = createWrapper();
        const { result } = renderHook(() => useListUsers(), { wrapper: Wrapper });

        await waitFor(() => expect(result.current.isError).toBe(true));
    });

    it('does not fetch when disabled', () => {
        // No server handler registered — any HTTP request would fail the test
        const { Wrapper } = createWrapper();
        const { result } = renderHook(() => useListUsers({ enabled: false }), { wrapper: Wrapper });

        expect(result.current.isFetching).toBe(false);
        expect(result.current.data).toBeUndefined();
    });
});
