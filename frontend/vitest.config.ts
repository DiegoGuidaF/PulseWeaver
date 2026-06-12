import { defineConfig } from 'vitest/config';
import react from '@vitejs/plugin-react';
import * as path from 'node:path';

export default defineConfig({
  plugins: [react()],
  test: {
    globals: true,
    environment: "happy-dom",
    setupFiles: ["./src/test/setup.ts"],
    css: true,
    // CI runners are far slower than dev machines; give tests extra headroom there.
    testTimeout: process.env.CI ? 15000 : 10000,
    pool: "forks",
    // On 4-vCPU CI runners, more workers just makes the heavy Mantine component
    // suites starve each other and trip per-test timeouts.
    maxWorkers: process.env.CI ? '50%' : '80%',
    // Console output from passing tests is pure noise in CI logs.
    silent: "passed-only",
    onConsoleLog(log) {
      // Mantine's focus-trap warning attaches the DOM node, which serializes to
      // thousands of lines under happy-dom; recharts warns about zero-size
      // containers on every chart render. Both bury real failures in the log.
      if (log.includes("[@mantine/hooks/use-focus-trap]")) return false;
      if (log.includes("The width(") && log.includes("of chart should be greater than 0")) return false;
    },
  },
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
});
