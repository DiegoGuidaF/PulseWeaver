import React from "react";
import ReactDOM from "react-dom/client";
import {
  QueryCache,
  QueryClient,
  QueryClientProvider,
} from "@tanstack/react-query";
import { ReactQueryDevtools } from "@tanstack/react-query-devtools";
import App from "./App";
import { getCurrentUserQueryKey } from "@/lib/api/@tanstack/react-query.gen";
import { toApiError } from "@/lib/api-client";
import "./lib/api-client/config"; // Initialize API client configuration
import "@mantine/core/styles.css";
import "@mantine/dates/styles.css";
import "@mantine/notifications/styles.css";
import "mantine-datatable/styles.layer.css";

// Helper to check if error is 401 and handle redirect
function handle401Error(error: unknown, isAuthMeQuery = false) {
  // Check if this is a 401 error
  // ApiError instances have a status property, or check the error object directly
  const { status } = toApiError(error);
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
      // Let useCurrentUser handle 401 gracefully (returns null)
      // ProtectedRoute will handle redirect based on isAuthenticated
      const currentUserKey = getCurrentUserQueryKey();
      const isAuthMeQuery =
        Array.isArray(query.queryKey) &&
        JSON.stringify(query.queryKey) === JSON.stringify(currentUserKey);
      // Only handle 401 for non-auth queries
      handle401Error(error, isAuthMeQuery);
    },
  }),
});

ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <QueryClientProvider client={queryClient}>
      <App />
      {import.meta.env.DEV && <ReactQueryDevtools initialIsOpen={false} />}
    </QueryClientProvider>
  </React.StrictMode>,
);
