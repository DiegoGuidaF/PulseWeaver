import { defineConfig } from 'vitest/config';
import react from '@vitejs/plugin-react';
import * as path from 'node:path';

export default defineConfig({
  plugins: [react()],
  test: {
    globals: true,
    environment: "happy-dom",
    globalSetup: ["./src/test/globalSetup.ts"],
    setupFiles: ["./src/test/setup.ts"],
    css: true,
    testTimeout: 15000,
    // No --localstorage-file: happy-dom supplies its own in-memory localStorage.
    // Passing a shared file path caused SQLite "database is locked" errors when
    // multiple Vitest workers initialised MSW's CookieStore concurrently.
  },
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
});
