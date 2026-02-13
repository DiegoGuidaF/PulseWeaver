import { ReactElement } from 'react';
import { render, RenderOptions } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter, MemoryRouterProps } from 'react-router-dom';

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
}

/**
 * Renders a component with all necessary providers (QueryClientProvider, MemoryRouter).
 * Useful for testing components that use React Query and React Router.
 *
 * @param ui - The component to render
 * @param options - Optional configuration
 * @param options.queryClient - Custom QueryClient instance (defaults to test-friendly config)
 * @param options.initialEntries - Initial router entries (defaults to ['/'])
 * @returns Render result with queryClient for test access
 */
export function renderWithProviders(
  ui: ReactElement,
  {
    queryClient = createTestQueryClient(),
    initialEntries = ['/'],
    ...renderOptions
  }: RenderWithProvidersOptions = {}
) {
  function Wrapper({ children }: { children: React.ReactNode }) {
    return (
      <QueryClientProvider client={queryClient}>
        <MemoryRouter initialEntries={initialEntries}>{children}</MemoryRouter>
      </QueryClientProvider>
    );
  }

  const result = render(ui, { wrapper: Wrapper, ...renderOptions });
  return {
    ...result,
    queryClient,
  };
}
