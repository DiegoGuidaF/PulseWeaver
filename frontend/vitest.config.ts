import { defineConfig } from 'vitest/config';
import react from '@vitejs/plugin-react';
import tailwindcss from '@tailwindcss/vite';
import * as path from 'node:path';

export default defineConfig({
  plugins: [react(), tailwindcss()],
  test: {
    globals: true,
    environment: "happy-dom",
    globalSetup: ["./src/test/globalSetup.ts"],
    setupFiles: ["./src/test/setup.ts"],
    css: true, // Process CSS for Tailwind
    execArgv: ["--localstorage-file=./tmp-localstorage"],
  },
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
});
