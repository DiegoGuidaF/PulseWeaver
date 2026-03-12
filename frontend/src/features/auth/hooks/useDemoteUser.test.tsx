import { describe, expect, it } from "vitest";
import { act, renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { http } from "msw";
import React from "react";
import { server } from "@/test/setup";
import { authHandlers, endpoints, responses } from "@/test/mocks/handlers";
import { createMockUser } from "@/test/mocks/data";
import { getCurrentUserQueryKey, listUsersQueryKey } from "@/lib/api/@tanstack/react-query.gen";
import { useDemoteUser } from "./useDemoteUser";

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

describe('useDemoteUser', () => {
    it('invalidates user list and current user on success', async () => {
        server.use(authHandlers.demoteUser.success());

        const { queryClient, Wrapper } = createWrapper();
        queryClient.setQueryData(listUsersQueryKey(), [createMockUser()]);
        queryClient.setQueryData(getCurrentUserQueryKey(), createMockUser());

        const { result } = renderHook(() => useDemoteUser(), { wrapper: Wrapper });

        act(() => {
            result.current.mutate({ path: { user_id: 2 } });
        });

        await waitFor(() => expect(result.current.isSuccess).toBe(true));

        expect(queryClient.getQueryState(listUsersQueryKey())?.isInvalidated).toBe(true);
        expect(queryClient.getQueryState(getCurrentUserQueryKey())?.isInvalidated).toBe(true);
    });

    it('enters error state on failure', async () => {
        server.use(
            http.post(endpoints.demoteUser, () =>
                responses.forbidden({ error: 'Forbidden role change' })
            )
        );

        const { Wrapper } = createWrapper();
        const { result } = renderHook(() => useDemoteUser(), { wrapper: Wrapper });

        act(() => {
            result.current.mutate({ path: { user_id: 1 } });
        });

        await waitFor(() => expect(result.current.isError).toBe(true));
    });
});
