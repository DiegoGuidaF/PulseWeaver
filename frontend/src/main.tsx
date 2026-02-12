import React from "react";
import ReactDOM from "react-dom/client";
import {
  MutationCache,
  QueryCache,
  QueryClient,
  QueryClientProvider,
} from "@tanstack/react-query";
import { ReactQueryDevtools } from "@tanstack/react-query-devtools";
import App from "./App";
import { queryKeys } from "@/lib/api-client";
import "./lib/api-client/config"; // Initialize API client configuration
import "./index.css";

// Helper to check if error is 401 and handle redirect
function handle401Error(error: unknown, isAuthMeQuery = false) {
  // Check if this is a 401 error
  // ApiError instances have a status property, or check the error object directly
  const status =
    error instanceof Error && "status" in error
      ? (error as any).status
      : (error as any)?.status;
  if (status === 401 && !isAuthMeQuery) {
    // Don't redirect if already on login page or if this is the auth/me query
    if (window.location.pathname !== "/login") {
      const returnTo = window.location.pathname + window.location.search;
      window.location.href = `/login?returnTo=${encodeURIComponent(returnTo)}`;
    }
  }
}

const queryClient = new QueryClient({
  queryCache: new QueryCache({
    onError: (error, query) => {
      // Don't redirect for /auth/me queries - 401 is expected when not logged in
      const isAuthMeQuery =
        query.queryKey[0] === queryKeys.auth.currentUser[0] &&
        query.queryKey[1] === queryKeys.auth.currentUser[1];
      handle401Error(error, isAuthMeQuery);
    },
  }),
  mutationCache: new MutationCache({
    onError: (error) => {
      // Handle 401 errors from mutations (but not logout/login mutations)
      // We check the mutation key if available, but for now handle all 401s
      handle401Error(error, false);
    },
  }),
});

ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <QueryClientProvider client={queryClient}>
      <App />
      <ReactQueryDevtools initialIsOpen={false} />
    </QueryClientProvider>
  </React.StrictMode>,
);
