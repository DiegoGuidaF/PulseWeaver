import { defineConfig, loadEnv } from 'vite'
import react from '@vitejs/plugin-react'
import checker from 'vite-plugin-checker'
import * as path from "node:path";

export default defineConfig(({ mode }) => {
  const rootDir = path.resolve(__dirname, '..');
  const env = loadEnv(mode, rootDir, '');
  const apiPort = env.SERVER_PORT || '8080';

  return {
    plugins: [
      react(),
      checker({ typescript: { tsconfigPath: './tsconfig.app.json' } }),
    ],
    resolve: {
      alias: {
        "@": path.resolve(__dirname, "./src"),
      },
    },
    server: {
      port: 5173,
      proxy: {
        // Proxy API requests to Go backend during development
        '/api': {
          target: `http://localhost:${apiPort}`,
          changeOrigin: true,
          secure: false,
        },
        '/health': {
          target: `http://localhost:${apiPort}`,
          changeOrigin: true,
        }
      }
    }
  };
})