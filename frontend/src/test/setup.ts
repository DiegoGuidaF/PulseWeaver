import '@testing-library/jest-dom';
import { beforeAll, afterEach, afterAll } from 'vitest';
import { setupServer } from 'msw/node';
import '@mantine/core/styles.css';
import '@mantine/notifications/styles.css';
import { defaultHandlers } from './mocks/handlers';

// Layer 1 — global happy-path defaults. Every test starts in an authenticated,
// data-loaded state without any per-test server.use() call.
export const server = setupServer(...defaultHandlers);

// Start server before all tests
beforeAll(() => {
    server.listen({ onUnhandledRequest: 'error' });
});

// Reset handlers after each test (restores layer 1 defaults)
afterEach(() => {
    server.resetHandlers();
});

// Clean up after all tests
afterAll(() => {
    server.close();
});
