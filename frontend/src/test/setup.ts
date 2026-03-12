import "@testing-library/jest-dom";
import { afterAll, afterEach, beforeAll } from "vitest";
import { setupServer } from "msw/node";

import "@mantine/core/styles.css";
import "@mantine/notifications/styles.css";
import { defaultHandlers } from "./mocks/handlers";
import { cleanup } from "@testing-library/react";
import { notifications } from "@mantine/notifications";

// 1. Polyfill localStorage and sessionStorage for Node 25 / MSW compatibility
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
