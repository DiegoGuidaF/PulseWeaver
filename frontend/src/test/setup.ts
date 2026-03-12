import '@testing-library/jest-dom';
import { beforeAll, afterEach, afterAll } from 'vitest';
import { setupServer } from 'msw/node';

// Node 25+ exposes a built-in `localStorage` global whose methods are stubs
// (they throw) unless `--localstorage-file` is passed to the worker process.
// Rather than sharing a SQLite file across workers (which causes "database is
// locked" errors when multiple workers start MSW's CookieStore concurrently),
// we install a per-worker in-memory implementation here.
if (typeof localStorage === 'undefined' || typeof localStorage.getItem !== 'function') {
    const store = new Map<string, string>();
    Object.defineProperty(globalThis, 'localStorage', {
        value: {
            getItem: (key: string) => store.get(key) ?? null,
            setItem: (key: string, value: string) => { store.set(key, String(value)); },
            removeItem: (key: string) => { store.delete(key); },
            clear: () => { store.clear(); },
            get length() { return store.size; },
            key: (index: number) => [...store.keys()][index] ?? null,
        },
        writable: true,
        configurable: true,
    });
}
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
