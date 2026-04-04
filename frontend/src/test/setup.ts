import "@testing-library/jest-dom";
import { afterAll, afterEach, beforeAll } from "vitest";
import { setupServer } from "msw/node";

import "@mantine/core/styles.css";
import "@mantine/notifications/styles.css";
import { defaultHandlers } from "./mocks/handlers";
import { cleanup } from "@testing-library/react";
import { notifications } from "@mantine/notifications";

// 1. Polyfill ResizeObserver — missing in happy-dom; required by mantine-datatable to render rows
// and by other Mantine layout components (SegmentedControl, Select, etc.).
//
// The callback fires asynchronously (via queueMicrotask) rather than synchronously during
// observe(). Synchronous firing causes an infinite re-render loop in components like
// SegmentedControl that update layout state in response to resize events — React's act()
// then waits forever for effects to settle. Async firing breaks the loop while still
// delivering the non-zero dimensions that mantine-datatable needs before waitFor times out.
globalThis.ResizeObserver = class ResizeObserver {
  private callback: ResizeObserverCallback;
  constructor(callback: ResizeObserverCallback) {
    this.callback = callback;
  }
  observe(target: Element) {
    // happy-dom returns 0x0 from getBoundingClientRect — provide a real size so
    // mantine-datatable renders rows instead of treating the container as invisible.
    const contentRect = { width: 1024, height: 768, top: 0, left: 0, bottom: 768, right: 1024, x: 0, y: 0 } as DOMRectReadOnly;
    queueMicrotask(() => {
      this.callback(
        [{ contentRect, target, borderBoxSize: [], contentBoxSize: [], devicePixelContentBoxSize: [] } as unknown as ResizeObserverEntry],
        this,
      );
    });
  }
  unobserve() {}
  disconnect() {}
};

// 2. Polyfill document.fonts — missing in happy-dom; required by Mantine Textarea autosize.
// Autosize.tsx calls document.fonts.addEventListener('loadingdone', ...) in a useEffect.
// Without this, opening any modal that contains a DeviceProfileCard (which has <Textarea autosize>)
// throws "Cannot read properties of undefined (reading 'addEventListener')".
if (!document.fonts) {
  Object.defineProperty(document, 'fonts', {
    value: {
      addEventListener: () => {},
      removeEventListener: () => {},
      dispatchEvent: () => false,
    },
    writable: true,
    configurable: true,
  });
}

// 3. Polyfill localStorage and sessionStorage for Node 25 / MSW compatibility
const createStorageMock = () => {
  let store: Record<string, string> = {};
  return {
    getItem: (key: string) => store[key] || null,
    setItem: (key: string, value: string) => {
      store[key] = value.toString();
    },
    removeItem: (key: string) => {
      delete store[key];
    },
    clear: () => {
      store = {};
    },
    get length() {
      return Object.keys(store).length;
    },
    key: (index: number) => Object.keys(store)[index] || null,
  };
};

Object.defineProperty(window, "localStorage", { value: createStorageMock() });
Object.defineProperty(window, "sessionStorage", { value: createStorageMock() });

// Global happy-path defaults. Every test starts in an authenticated,
// data-loaded state without any per-test server.use() call.
export const server = setupServer(...defaultHandlers);

// Start server before all tests
beforeAll(() => {
  server.listen({ onUnhandledRequest: "error" });
});

// Reset handlers after each test (restores layer 1 defaults)
afterEach(() => {
  cleanup();
  server.resetHandlers();
  notifications.clean();
  notifications.cleanQueue();
  window.localStorage.clear();
  window.sessionStorage.clear();
});

// Clean up after all tests
afterAll(() => {
  server.close();
});
