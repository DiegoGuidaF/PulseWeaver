import type { ReactElement } from 'react';
import { render, type RenderOptions } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import {
    createMemoryRouter,
    RouterProvider,
    type MemoryRouterProps,
} from 'react-router-dom';
import { MantineProvider } from '@mantine/core';
import { ModalsProvider } from '@mantine/modals';
import { Notifications } from '@mantine/notifications';
import { DatesProvider } from '@mantine/dates';
import { DateTimePrefsProvider } from '@/contexts/DateTimePrefsContext';

/**
 * userEvent session for component tests — the single place to tune session
 * options (delay, pointer-events checks) for the whole suite. Use this instead
 * of calling userEvent.setup() directly.
 */
export function setupUser() {
    return userEvent.setup();
}

/**
 * Creates a test-friendly QueryClient with retry disabled for faster test failures.
 */
function createTestQueryClient() {
    return new QueryClient({
        defaultOptions: {
            queries: {
                retry: false,
            },
            mutations: {
                retry: false,
            },
        },
    });
}

interface RenderWithProvidersOptions extends Omit<RenderOptions, 'wrapper'> {
    queryClient?: QueryClient;
    initialEntries?: MemoryRouterProps['initialEntries'];
    path?: string;
}

/**
 * Renders a component with all necessary providers.
 * Wraps with MantineProvider, Notifications, ModalsProvider, DateTimePrefsProvider,
 * DatesProvider, QueryClientProvider, and a memory data router.
 *
 * A data router (`createMemoryRouter`) is used so components relying on `useBlocker`
 * (e.g. via `useUnsavedChangesGuard`) work in tests without being mocked.
 *
 * @param ui - The component to render
 * @param options - Optional configuration
 * @param options.queryClient - Custom QueryClient instance (defaults to test-friendly config)
 * @param options.initialEntries - Initial router entries (defaults to ['/'])
 * @param options.path - Optional route path wrapper for components using useParams
 * @returns Render result with queryClient for test access
 */
export function renderWithProviders(
    ui: ReactElement,
    {
        queryClient = createTestQueryClient(),
        initialEntries = ['/'],
        path,
        ...renderOptions
    }: RenderWithProvidersOptions = {}
) {
    const router = createMemoryRouter(
        path
            ? [
                  { path, element: ui },
                  { path: '*', element: null },
              ]
            : [{ path: '*', element: ui }],
        { initialEntries }
    );

    // env="test" disables Mantine transitions and portals so popovers/drawers
    // mount synchronously — interactions stay deterministic on slow CI runners.
    const result = render(
        <MantineProvider env="test">
            <Notifications />
            <DateTimePrefsProvider>
                <DatesProvider settings={{ locale: 'en' }}>
                    <QueryClientProvider client={queryClient}>
                        <ModalsProvider>
                            <RouterProvider router={router} />
                        </ModalsProvider>
                    </QueryClientProvider>
                </DatesProvider>
            </DateTimePrefsProvider>
        </MantineProvider>,
        renderOptions
    );
    return {
        ...result,
        queryClient,
    };
}
