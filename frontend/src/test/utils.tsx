import type { ReactElement } from 'react';
import { render, type RenderOptions } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import {
    MemoryRouter,
    Route,
    Routes,
    type MemoryRouterProps,
} from 'react-router-dom';
import { MantineProvider } from '@mantine/core';
import { Notifications } from '@mantine/notifications';
import { DatesProvider } from '@mantine/dates';
import { DateTimePrefsProvider } from '@/contexts/DateTimePrefsContext';

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
 * Wraps with MantineProvider, Notifications, DateTimePrefsProvider, DatesProvider,
 * QueryClientProvider, and MemoryRouter.
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
    const content = path ? (
        <Routes>
            <Route path={path} element={ui} />
            <Route path="*" element={null} />
        </Routes>
    ) : (
        ui
    );

    function Wrapper({ children }: { children: React.ReactNode }) {
        return (
            <MantineProvider>
                <Notifications />
                <DateTimePrefsProvider>
                    <DatesProvider settings={{ locale: 'en' }}>
                        <QueryClientProvider client={queryClient}>
                            <MemoryRouter initialEntries={initialEntries}>
                                {children}
                            </MemoryRouter>
                        </QueryClientProvider>
                    </DatesProvider>
                </DateTimePrefsProvider>
            </MantineProvider>
        );
    }

    const result = render(content, { wrapper: Wrapper, ...renderOptions });
    return {
        ...result,
        queryClient,
    };
}
