import {describe, expect, it} from 'vitest';
import {screen, waitFor} from '@testing-library/react';
import {delay} from 'msw';
import {server} from '@/test/setup';
import {renderWithProviders} from '@/test/utils';
import {ProtectedRoute} from './ProtectedRoute';
import {AuthProvider} from '@/contexts/AuthContext';
import React from "react";
import {handlers, responses} from "@/test/mocks/handlers.ts";
import {createMockUser} from "@/test/mocks/data.ts";

function renderProtectedRoute(children: React.ReactNode, initialEntries = ['/protected']) {
    return renderWithProviders(
        <AuthProvider>
            <ProtectedRoute>{children}</ProtectedRoute>
        </AuthProvider>,
        {initialEntries}
    );
}

describe('ProtectedRoute', () => {
    it('shows loading state while checking auth', () => {
        server.use(
            handlers.auth.meHandler(undefined, async () => {
                await delay('infinite');
                return responses.ok(createMockUser());
            })
        );

        renderProtectedRoute(<div>Protected Content</div>);

        expect(screen.getByText(/loading/i)).toBeInTheDocument();
        expect(screen.queryByText('Protected Content')).not.toBeInTheDocument();
    });

    it('redirects to login when not authenticated', async () => {
        server.use(
            handlers.auth.meHandler(undefined, async () => {
                return responses.unauthorized()
            })
        );

        renderProtectedRoute(<div>Protected Content</div>, ['/devices']);

        // Wait for redirect - ProtectedRoute renders Navigate component
        // Protected content should not be visible
        await waitFor(() => {
            expect(screen.queryByText('Protected Content')).not.toBeInTheDocument();
        });
    });

    it('renders children when authenticated', async () => {
        server.use(
            handlers.auth.meHandler()
        );

        renderProtectedRoute(<div>Protected Content</div>);

        // Wait for auth check to complete
        await waitFor(() => {
            expect(screen.getByText('Protected Content')).toBeInTheDocument();
        });

        // Should not show loading or login form
        expect(screen.queryByText(/loading/i)).not.toBeInTheDocument();
        expect(screen.queryByLabelText(/username/i)).not.toBeInTheDocument();
    });
});
